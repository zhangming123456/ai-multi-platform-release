package services

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
)

func FindInspectionMaterial(id string) (*models.InspectionMaterial, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("素材不存在")
	}
	var material models.InspectionMaterial
	err := GetOrm().QueryTable(new(models.InspectionMaterial)).
		Filter("id", id).
		Filter("is_active", true).
		One(&material)
	if err != nil {
		return nil, errors.New("素材不存在")
	}
	return &material, nil
}

func SaveInspectionMaterial(material *models.InspectionMaterial) error {
	o := GetOrm()
	now := time.Now()
	if material.ID == "" {
		material.ID = uuid.NewString()
		material.IsActive = true
		material.CreatedAt = now
		material.UpdatedAt = now
		_, err := o.Insert(material)
		return err
	}
	material.UpdatedAt = now
	_, err := o.Update(material)
	return err
}

func DeleteInspectionMaterial(id string) error {
	material, err := FindInspectionMaterial(id)
	if err != nil {
		return err
	}
	material.IsActive = false
	material.UpdatedAt = time.Now()
	_, err = GetOrm().Update(material)
	return err
}

// SeedInspectionMaterialsBulk 批量创建 20 条检查项素材，用于测试数据初始化
func SeedInspectionMaterialsBulk() error {
	o := GetOrm()
	specs := []struct {
		category  string
		title     string
		standard  string
		scoreType string
		maxScore  int
	}{
		// 形象类
		{category: "形象", title: "门头招牌完整度", standard: "门店招牌无破损缺字、夜间亮灯正常、无明显污渍", scoreType: "score", maxScore: 5},
		{category: "形象", title: "橱窗 POP 规范", standard: "橱窗 POP 张贴平整、内容对应档期、无翘边无过期", scoreType: "score", maxScore: 5},
		{category: "形象", title: "灯箱画面完好", standard: "灯箱画面无折痕无褪色、光源均匀、无暗区", scoreType: "pass_fail", maxScore: 5},
		{category: "形象", title: "入口地垫整洁", standard: "地垫铺设平整、无异味无污渍、定期更换清洗", scoreType: "pass_fail", maxScore: 5},

		// 卫生类
		{category: "卫生", title: "地面清洁度", standard: "地面无脚印、无垃圾、无水渍、角落无积尘", scoreType: "score", maxScore: 5},
		{category: "卫生", title: "玻璃镜面清洁", standard: "玻璃无指纹无水印、镜面透亮、胶条完好", scoreType: "score", maxScore: 5},
		{category: "卫生", title: "洗手间清洁", standard: "洗手间无异味、洗手台干爽、设备完好、消耗品充足", scoreType: "pass_fail", maxScore: 5},
		{category: "卫生", title: "仓库卫生", standard: "仓库地面无垃圾、货架无积尘、无过期临期商品", scoreType: "pass_fail", maxScore: 5},

		// 陈列类
		{category: "陈列", title: "橱窗模特搭配", standard: "模特搭配符合当季主题、配饰完整、价格标签明显", scoreType: "score", maxScore: 5},
		{category: "陈列", title: "主推陈列区", standard: "主推陈列区饱满、价格签对齐、有 POP 或立牌", scoreType: "score", maxScore: 5},
		{category: "陈列", title: "价签规范度", standard: "所有陈列商品一一对应价签、价签端正清晰", scoreType: "pass_fail", maxScore: 5},
		{category: "陈列", title: "叠装/挂装规范", standard: "叠装整齐无褶皱、挂装方向一致间距均匀", scoreType: "pass_fail", maxScore: 5},

		// 服务类
		{category: "服务", title: "员工着装规范", standard: "员工身着统一工服、工牌佩戴正确、发型妆容合规", scoreType: "score", maxScore: 5},
		{category: "服务", title: "顾客接待礼仪", standard: "顾客进店 3 秒内问候、距离 1.5 米微笑、主动引导", scoreType: "score", maxScore: 5},
		{category: "服务", title: "收银唱收唱付", standard: "收银时唱收唱付、双手递接小票和商品、微笑道别", scoreType: "pass_fail", maxScore: 5},
		{category: "服务", title: "客服处理时效", standard: "顾客投诉 10 分钟内响应处理、记录完整跟进", scoreType: "pass_fail", maxScore: 5},

		// 安全类
		{category: "安全", title: "灭火器状态", standard: "灭火器在有效期内、铅封完好、压力指示正常", scoreType: "score", maxScore: 5},
		{category: "安全", title: "消防通道畅通", standard: "消防通道无商品杂物堆放、疏散指示牌完好", scoreType: "pass_fail", maxScore: 5},
		{category: "安全", title: "应急照明测试", standard: "应急照明灯完好、断电后可正常点亮 30 分钟以上", scoreType: "pass_fail", maxScore: 5},

		// 员工管理类
		{category: "员工管理", title: "考勤与班次", standard: "全员按排班到岗、考勤记录完整、无无故缺勤", scoreType: "pass_fail", maxScore: 5},
	}
	for _, s := range specs {
		// 同分类+同标题的素材已存在则跳过，避免重复写入
		existCount, _ := o.QueryTable(new(models.InspectionMaterial)).
			Filter("category", s.category).
			Filter("title", s.title).
			Filter("is_active", true).
			Count()
		if existCount > 0 {
			continue
		}
		m := &models.InspectionMaterial{
			ID:           uuid.NewString(),
			Category:     s.category,
			Title:        s.title,
			Standard:     s.standard,
			ScoreType:    s.scoreType,
			MaxScore:     s.maxScore,
			ScoreOptions: models.MarshalScoreOptions(models.DefaultScoreOptions(s.scoreType, s.maxScore)),
			IsActive:     true,
		}
		if _, err := o.Insert(m); err != nil {
			return err
		}
	}
	return nil
}
