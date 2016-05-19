// Copyright 2016 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ingore

//
// 数据来源:
//	https://github.com/langhua9527/BlackheartedHospital
//

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"unicode/utf8"

	hospital_parser "github.com/chai2010/ptyy/internal/hospital"
	"github.com/chai2010/ptyy/internal/static"
)

func main() {
	infoList := parseInfoList()
	genListFile("z_list.go", infoList)
	genPinyinFile("z_pinyin.go", infoList, parsePinyinFile())
}

type HospitalInfo struct {
	Name    string   // 名称
	City    string   // 城市
	Owner   []string // 投资者
	Comment []string // 注释
}

// 按unicode排序
type byHospitalInfo []HospitalInfo

func (d byHospitalInfo) Len() int           { return len(d) }
func (d byHospitalInfo) Less(i, j int) bool { return d[i].Name < d[j].Name }
func (d byHospitalInfo) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }

func genListFile(filename string, infoList []HospitalInfo) {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, `
// Copyright 2016 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// generated by go run gen_helper.go; DO NOT EDIT!!!

package ptyy
`[1:])

	fmt.Fprintf(&buf, "// 共 %d 个记录\n", len(infoList))
	fmt.Fprintln(&buf, `var _AllHospitalInfoList = []HospitalInfo{`)
	for _, info := range infoList {
		fmt.Fprintf(&buf, "{Name: %q, City: %q},\n", info.Name, info.City)
	}
	fmt.Fprintln(&buf, "}")

	data, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// 生成拼音表格
func genPinyinFile(filename string, infoList []HospitalInfo, pyMap map[rune][]string) {
	// 统计出现的汉字列表
	var usedRuneMap = make(map[rune]bool)
	for _, info := range infoList {
		for _, r := range info.Name {
			usedRuneMap[r] = true
		}
		for _, r := range info.City {
			usedRuneMap[r] = true
		}
	}

	var buf bytes.Buffer

	fmt.Fprintln(&buf, `
// Copyright 2016 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// generated by go run gen_helper.go; DO NOT EDIT!!!

package ptyy
`[1:])

	fmt.Fprintln(&buf, `var _RunePinyinTable = map[rune]string{`)
	for r, pyList := range pyMap {
		// 只输出用到的汉字
		if !usedRuneMap[r] {
			continue
		}

		// 多音字读音选择补丁
		if pyPatch := g_pinyinPatch[r]; pyPatch != "" {
			pyList = []string{pyPatch}
		}

		// 多音字需要手工处理
		if len(pyList) > 1 {
			fmt.Printf("多音字: '%s': %q\n", string(r), pyList)
			continue
		}

		fmt.Fprintf(&buf, "'%s': %q,\n", string(r), pyList[0])
	}
	fmt.Fprintln(&buf, "}")

	var data = buf.Bytes()
	var err error

	data, err = format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// 读取拼音数据(有多音字)
func parsePinyinFile() map[rune][]string {
	data, err := ioutil.ReadFile("./internal/static/pinyin1234.txt")
	if err != nil {
		log.Fatal(err)
	}

	// 单字母和多字母拼音需要特殊处理
	runePinyinMap := make(map[rune][]string)

	// 分析行信息
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		curLine := strings.TrimSpace(lines[i])

		// 分析行
		secList := strings.Split(curLine, " ")
		if len(secList) < 2 {
			continue
		}

		// 第一个是拼音, 当个字母单独处理
		curPinyin := strings.TrimSpace(secList[0])
		for i := 1; i < len(secList); i++ {
			curSec := strings.TrimSpace(secList[i])
			r := ([]rune(curSec))[0]
			if _, ok := runePinyinMap[r]; ok {
				runePinyinMap[r] = append(runePinyinMap[r], curPinyin)
			} else {
				runePinyinMap[r] = []string{curPinyin}
			}
		}
	}

	// 将拼音排序
	// 剔除重复的当字符拼音
	for k, pyList := range runePinyinMap {
		sort.Strings(pyList)
		for i := 0; i < len(pyList)-1; i++ {
			if len(pyList[i]) == 1 && pyList[i][0] == pyList[i+1][0] {
				pyList = append(pyList[:i], pyList[i+1:]...)
				i--
			}
		}
		runePinyinMap[k] = pyList
	}
	return runePinyinMap
}

// 读取列表文件
func parseInfoList() (infoList []HospitalInfo) {
	r := strings.NewReader(static.Files["hospital_list.20160508.json"])
	db, err := hospital_parser.ReadJsonFrom(r)
	if err != nil {
		log.Fatal(err)
	}
	for _, info := range db {
		infoList = append(infoList, HospitalInfo{
			Name: info.Name,
			City: info.City,
		})
	}
	// 输出
	sort.Sort(byHospitalInfo(infoList))
	return
}

// 是否为忽略的行
func isIngoreLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return true
	}
	for _, key := range g_ingoreLineKeywordsList {
		if strings.Contains(line, key) {
			return true
		}
	}
	return false
}

// 是否是城市名
func isCityName(name string) bool {
	name = strings.TrimSpace(name)
	for s, _ := range g_CityMap {
		if s == name {
			return true
		}
	}
	return false
}

// 公司名
func isCompanyName(name string) bool {
	if utf8.RuneCountInString(name) >= 20 {
		return false
	}
	for _, key := range g_CompanyNameKeywordsList {
		if strings.Contains(name, key) {
			return true
		}
	}
	return false
}

// 是否为医院名(`-`开头)
func isHospitalName(name string) bool {
	for _, key := range g_HospitalKeywordsList {
		if strings.Contains(name, key) {
			return true
		}
	}
	return false
}

// 汉字读音字补丁(回避多音字问题)
var g_pinyinPatch = map[rune]string{
	'乐': "le4",
	'拉': "la1",
	'重': "chong2",
	'漯': "luo4",
	'寿': "shou4",
	'创': "chuang4",
	'沙': "sha1",
	'玛': "ma3",
	'脉': "mai4",
	'朝': "chao2",
	'宿': "su4",
	'咽': "yan1",
	'番': "pan1",
	'都': "du1",
	'厦': "xia4",
	'结': "jie2",
	'脊': "ji2",
	'伯': "bo2",
	'似': "si4",
}

// 忽略行的关键字
var g_ingoreLineKeywordsList = []string{
	"BlackheartedHospital",
	"最新补充",
	"欢迎更新",
	"版本",
	"- 1.",
	"- 2.",
	"- 3.",
	"- 4.",
	"- 5.",
	"- 6.",
	"- 7.",
	"- 8.",
	"- 9.",
	"这里的绝大部分军队医院治疗妇科病",
}

// 医院名关键字
var g_HospitalKeywordsList = []string{
	"医院",
	"医疗中心",
	"中医",
	"妇科",
	"门诊部",
	"美容诊所",
	"五官中心",
	"北京天院",
	"新医科",
	"眼科中心",
	"产科中心",
	"体检中心",
	"前列腺专科",
	"长征院",
	"长征医院",
	"心理院",
	"保健中心院",
}

// 公司名关键字
var g_CompanyNameKeywordsList = []string{
	"公司",
	"生态园",
	"研究所",
	"全资机构",
	"整形网",
	"不育网",
	"肿瘤网",
}

// 城市名列表
var g_CityMap = map[string]bool{
	"上海":   true,
	"北京":   true,
	"苏州":   true,
	"天津":   true,
	"广州":   true,
	"珠海":   true,
	"惠州":   true,
	"中山":   true,
	"汕头":   true,
	"东莞":   true,
	"江门":   true,
	"肇庆":   true,
	"佛山":   true,
	"深圳":   true,
	"昆明":   true,
	"玉溪":   true,
	"曲靖":   true,
	"重庆":   true,
	"成都":   true,
	"雅安":   true,
	"遵义":   true,
	"凉山":   true,
	"南充":   true,
	"乐山":   true,
	"福州":   true,
	"舟山":   true,
	"厦门":   true,
	"莆田":   true,
	"宁波":   true,
	"杭州":   true,
	"湖州":   true,
	"泉州":   true,
	"金华":   true,
	"嘉兴":   true,
	"台州":   true,
	"温州":   true,
	"龙岩":   true,
	"济南":   true,
	"潍坊":   true,
	"青岛":   true,
	"德州":   true,
	"威海":   true,
	"聊城":   true,
	"淄博":   true,
	"哈尔滨":  true,
	"长春":   true,
	"四平":   true,
	"延边":   true,
	"沈阳":   true,
	"大连":   true,
	"无锡":   true,
	"南京":   true,
	"张家港":  true,
	"泰州":   true,
	"盐城":   true,
	"宿迁":   true,
	"淮安":   true,
	"南通":   true,
	"武汉":   true,
	"荆州":   true,
	"黄冈":   true,
	"黄石":   true,
	"襄阳":   true,
	"乌海":   true,
	"呼和浩特": true,
	"贵阳":   true,
	"铜仁":   true,
	"安顺":   true,
	"毕节":   true,
	"长沙":   true,
	"郴州":   true,
	"湘潭":   true,
	"娄底":   true,
	"南昌":   true,
	"九江":   true,
	"吉安":   true,
	"萍乡":   true,
	"赣州":   true,
	"上饶":   true,
	"太原":   true,
	"临汾":   true,
	"阳泉":   true,
	"长治":   true,
	"大同":   true,
	"晋城":   true,
	"晋中":   true,
	"运城":   true,
	"西安":   true,
	"包头":   true,
	"蚌埠":   true,
	"亳州":   true,
	"芜湖":   true,
	"巢湖":   true,
	"淮北":   true,
	"合肥":   true,
	"安阳":   true,
	"郑州":   true,
	"许昌":   true,
	"廊坊":   true,
	"保定":   true,
	"唐山":   true,
	"洛阳":   true,
	"信阳":   true,
	"平顶山":  true,
	"漯河":   true,
	"石家庄":  true,
	"邯郸":   true,
	"拉萨":   true,
	"银川":   true,
	"兰州":   true,
	"桂林":   true,
	"柳州":   true,
	"伊犁":   true,
	"伊宁":   true,
	"乌鲁木齐": true,
	"海口":   true,
}
