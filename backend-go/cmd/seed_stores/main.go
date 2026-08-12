package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/services"
)

var cities = []struct {
	Name      string
	Districts []string
	Streets   []string
}{
	{Name: "北京", Districts: []string{"朝阳区", "海淀区", "丰台区", "东城区", "西城区", "通州区", "大兴区", "石景山区"}, Streets: []string{"建国路", "中关村大街", "望京街", "三里屯路", "五道口", "国贸", "王府井", "西单"}},
	{Name: "上海", Districts: []string{"浦东新区", "徐汇区", "静安区", "黄浦区", "长宁区", "杨浦区", "虹口区", "闵行区"}, Streets: []string{"南京路", "淮海中路", "陆家嘴", "徐家汇", "静安寺", "人民广场", "五角场", "虹桥路"}},
	{Name: "广州", Districts: []string{"天河区", "越秀区", "海珠区", "白云区", "番禺区", "荔湾区", "黄埔区"}, Streets: []string{"体育西路", "北京路", "珠江新城", "中山大道", "江南西", "上下九", "龙洞"}},
	{Name: "深圳", Districts: []string{"南山区", "福田区", "罗湖区", "宝安区", "龙岗区", "龙华区", "光明区"}, Streets: []string{"科技园路", "华强北", "东门", "海岸城", "车公庙", "坂田", "西丽"}},
	{Name: "杭州", Districts: []string{"西湖区", "拱墅区", "上城区", "滨江区", "余杭区", "萧山区"}, Streets: []string{"武林路", "湖滨路", "文三路", "阿里巴巴园区", "西溪路", "钱江路"}},
	{Name: "成都", Districts: []string{"锦江区", "武侯区", "青羊区", "金牛区", "高新区", "成华区"}, Streets: []string{"春熙路", "天府大道", "宽窄巷子", "太古里", "建设路", "科华北路"}},
	{Name: "武汉", Districts: []string{"武昌区", "洪山区", "江汉区", "江岸区", "汉阳区", "硚口区"}, Streets: []string{"光谷大道", "江汉路", "街道口", "楚河汉街", "中南路", "徐东大街"}},
	{Name: "南京", Districts: []string{"鼓楼区", "玄武区", "秦淮区", "建邺区", "江宁区", "栖霞区"}, Streets: []string{"新街口", "夫子庙", "湖南路", "河西大街", "百家湖", "仙林大道"}},
	{Name: "重庆", Districts: []string{"渝中区", "江北区", "南岸区", "渝北区", "沙坪坝区", "九龙坡区"}, Streets: []string{"解放碑", "观音桥", "南滨路", "三峡广场", "杨家坪", "大学城"}},
	{Name: "苏州", Districts: []string{"姑苏区", "吴中区", "工业园区", "虎丘区", "相城区", "吴江区"}, Streets: []string{"观前街", "平江路", "金鸡湖大道", "狮山路", "独墅湖", "竹园路"}},
	{Name: "西安", Districts: []string{"雁塔区", "碑林区", "未央区", "莲湖区", "新城区", "长安区"}, Streets: []string{"钟楼", "小寨", "大雁塔", "高新区路", "曲江路", "凤城路"}},
	{Name: "天津", Districts: []string{"和平区", "南开区", "河西区", "河东区", "河北区", "滨海新区"}, Streets: []string{"滨江道", "天津之眼", "五大道", "意式风情区", "开发区路", "津湾广场"}},
	{Name: "长沙", Districts: []string{"岳麓区", "芙蓉区", "天心区", "开福区", "雨花区"}, Streets: []string{"五一广场", "黄兴路", "橘子洲头", "岳麓山", "梅溪湖路"}},
	{Name: "郑州", Districts: []string{"金水区", "二七区", "中原区", "管城区", "郑东新区"}, Streets: []string{"二七广场", "花园路", "CBD商务区", "紫荆山路", "东风路"}},
	{Name: "青岛", Districts: []string{"市南区", "市北区", "崂山区", "李沧区", "城阳区"}, Streets: []string{"中山路", "五四广场", "海尔路", "香港中路", "台东"}},
	{Name: "大连", Districts: []string{"中山区", "西岗区", "沙河口区", "甘井子区", "高新园区"}, Streets: []string{"青泥洼桥", "星海广场", "西安路", "软件园路", "开发区"}},
	{Name: "厦门", Districts: []string{"思明区", "湖里区", "集美区", "海沧区", "同安区"}, Streets: []string{"中山路", "曾厝垵", "SM城市广场", "嘉禾路", "软件园二期"}},
	{Name: "合肥", Districts: []string{"蜀山区", "包河区", "庐阳区", "瑶海区", "滨湖新区"}, Streets: []string{"淮河路", "天鹅湖路", "三里庵", "滨湖世纪城", "政务区路"}},
	{Name: "昆明", Districts: []string{"五华区", "盘龙区", "官渡区", "西山区", "呈贡区"}, Streets: []string{"南屏街", "翠湖路", "滇池路", "东风广场", "大学城路"}},
	{Name: "福州", Districts: []string{"鼓楼区", "台江区", "仓山区", "晋安区", "马尾区"}, Streets: []string{"东街口", "五四路", "中亭街", "宝龙城市广场", "金山大道"}},
	{Name: "济南", Districts: []string{"历下区", "市中区", "槐荫区", "天桥区", "历城区"}, Streets: []string{"泉城路", "经十路", "大明湖路", "洪家楼", "高新万达"}},
	{Name: "南宁", Districts: []string{"青秀区", "兴宁区", "西乡塘区", "良庆区", "江南区"}, Streets: []string{"朝阳路", "民族大道", "东盟商务区", "五象新区", "中山路"}},
	{Name: "南昌", Districts: []string{"东湖区", "西湖区", "青云谱区", "青山湖区", "红谷滩区"}, Streets: []string{"中山路", "八一大道", "红谷滩万达", "高新大道", "秋水广场"}},
	{Name: "贵阳", Districts: []string{"云岩区", "南明区", "花溪区", "观山湖区", "乌当区"}, Streets: []string{"中华路", "喷水池", "花果园", "金融城", "会展城"}},
	{Name: "太原", Districts: []string{"迎泽区", "杏花岭区", "万柏林区", "小店区", "尖草坪区"}, Streets: []string{"柳巷", "长风街", "亲贤街", "学府街", "龙城大街"}},
	{Name: "哈尔滨", Districts: []string{"南岗区", "道里区", "道外区", "香坊区", "松北区"}, Streets: []string{"中央大街", "秋林", "果戈里大街", "哈西万达", "松北大道"}},
}

var storePrefixes = []string{"旗舰店", "标准店", "社区店", "精品店", "体验店", "快闪店", "主力店", "形象店"}

func pinyinAbbr(city string) string {
	abbrs := map[string]string{
		"北京": "BJ", "上海": "SH", "广州": "GZ", "深圳": "SZ", "杭州": "HZ",
		"成都": "CD", "武汉": "WH", "南京": "NJ", "重庆": "CQ", "苏州": "SZ1",
		"西安": "XA", "天津": "TJ", "长沙": "CS", "郑州": "ZZ", "青岛": "QD",
		"大连": "DL", "厦门": "XM", "合肥": "HF", "昆明": "KM", "福州": "FZ",
		"济南": "JN", "南宁": "NN", "南昌": "NC", "贵阳": "GY", "太原": "TY",
		"哈尔滨": "HEB",
	}
	if abbr, ok := abbrs[city]; ok {
		return abbr
	}
	return city[:2]
}

func main() {
	if err := services.InitDatabase("app.db"); err != nil {
		fmt.Println("初始化数据库失败:", err)
		return
	}

	o := services.GetOrm()

	count, _ := o.QueryTable("stores").Count()
	if int(count) >= 2000 {
		fmt.Printf("已有 %d 条门店数据，跳过\n", count)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	surnames := []string{"张", "王", "李", "赵", "陈", "杨", "黄", "周", "吴", "徐", "孙", "马", "朱", "胡", "郭", "何", "林", "罗", "高", "梁"}
	givenNames := []string{"伟", "芳", "娜", "敏", "静", "丽", "强", "磊", "洋", "勇", "军", "杰", "秀英", "秀兰", "桂英", "建华", "建国", "建军", "志强", "志明", "文博", "浩然", "子涵", "雨桐"}

	batchSize := 500
	total := 2000
	now := time.Now()

	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}

		placeholders := make([]string, 0, end-start)
		args := make([]interface{}, 0, (end-start)*9)

		for i := start; i < end; i++ {
			id := uuid.NewString()
			cityIdx := rng.Intn(len(cities))
			city := cities[cityIdx]
			districtIdx := rng.Intn(len(city.Districts))
			district := city.Districts[districtIdx]
			streetIdx := rng.Intn(len(city.Streets))
			street := city.Streets[streetIdx]
			prefixIdx := rng.Intn(len(storePrefixes))
			prefix := storePrefixes[prefixIdx]

			storeNum := i + 1
			name := fmt.Sprintf("%s%s%s%d号%s", city.Name, district, street, rng.Intn(500)+1, prefix)
			code := fmt.Sprintf("%s-%03d-%05d", pinyinAbbr(city.Name), districtIdx+1, storeNum)
			address := fmt.Sprintf("%s%s%s%d号", city.Name, district, street, rng.Intn(500)+1)
			surname := surnames[rng.Intn(len(surnames))]
			given := givenNames[rng.Intn(len(givenNames))]
			contact := surname + given
			phone := fmt.Sprintf("1%d%08d", 3+rng.Intn(7), rng.Intn(100000000))

			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, 'active', ?, ?)")
			args = append(args, id, name, code, address, contact, phone, now, now)
		}

		sql := "INSERT INTO stores (id, name, code, address, contact, phone, status, created_at, updated_at) VALUES " + strings.Join(placeholders, ", ")
		if _, err := o.Raw(sql, args...).Exec(); err != nil {
			fmt.Println("插入失败:", err)
			return
		}
		fmt.Printf("已插入 %d / %d 条\n", end, total)
	}

	fmt.Println("2000 家门店数据生成完毕")
}
