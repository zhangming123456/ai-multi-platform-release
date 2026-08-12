package services

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
)

func SeedInspectionTemplates() error {
	o := GetOrm()
	count, err := o.QueryTable(new(models.InspectionTemplate)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	template := &models.InspectionTemplate{
		ID:          uuid.NewString(),
		Name:        "标准门店巡检",
		Description: "覆盖门店形象、卫生、陈列、服务、安全等常规检查项，适用于日常巡店",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	now := time.Now()
	items := []struct {
		Category      string
		Title         string
		Standard      string
		ScoreType     string
		MaxScore      int
		ScoreOptions  []models.ScoreOption
		RequireRemark bool
		RequirePhoto  bool
		ShowRemark    bool
		ShowPhoto     bool
	}{
		{Category: "形象", Title: "门头形象", Standard: "门店招牌完整、干净、夜间亮灯正常，无破损缺字", ScoreType: "score", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("score", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
		{Category: "卫生", Title: "环境卫生", Standard: "地面、柜台、货架无灰尘污渍，垃圾及时清理", ScoreType: "score", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("score", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
		{Category: "陈列", Title: "商品陈列", Standard: "商品摆放整齐、标签清晰、无过期临期商品混放", ScoreType: "score", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("score", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
		{Category: "服务", Title: "服务规范", Standard: "员工着装统一、佩戴工牌、服务用语规范", ScoreType: "pass_fail", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("pass_fail", 10), RequireRemark: true, RequirePhoto: false, ShowRemark: true, ShowPhoto: true},
		{Category: "安全", Title: "消防安全", Standard: "灭火器在有效期内、消防通道畅通无堵塞", ScoreType: "pass_fail", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("pass_fail", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
	}
	entries := make([]*models.InspectionTemplateItem, 0, len(items))
	for i, item := range items {
		entries = append(entries, &models.InspectionTemplateItem{
			ID:            uuid.NewString(),
			TemplateID:    template.ID,
			Category:      item.Category,
			Title:         item.Title,
			Standard:      item.Standard,
			ScoreType:     item.ScoreType,
			MaxScore:      item.MaxScore,
			ScoreOptions:  models.MarshalScoreOptions(item.ScoreOptions),
			RequireRemark: item.RequireRemark,
			RequirePhoto:  item.RequirePhoto,
			ShowRemark:    item.ShowRemark,
			ShowPhoto:     item.ShowPhoto,
			SortOrder:     i,
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Insert(template); err != nil {
		tx.Rollback()
		return err
	}
	for _, e := range entries {
		if _, err := tx.Insert(e); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func GetInspectionTemplateItems(templateID string) ([]models.InspectionTemplateItem, error) {
	var items []models.InspectionTemplateItem
	_, err := GetOrm().QueryTable(new(models.InspectionTemplateItem)).
		Filter("template_id", templateID).
		Filter("is_active", true).
		OrderBy("sort_order").
		All(&items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func EffectiveScoreOptions(item *models.InspectionTemplateItem) []models.ScoreOption {
	options := models.UnmarshalScoreOptions(item.ScoreOptions)
	if len(options) == 0 {
		return models.DefaultScoreOptions(item.ScoreType, item.MaxScore)
	}
	return options
}

func ValidateScoreOptions(options []models.ScoreOption) error {
	if len(options) == 0 {
		return errors.New("评分选项不能为空")
	}
	allowed := map[float64]bool{
		0: true, 0.5: true, 1: true, 2: true, 3: true, 4: true, 5: true,
	}
	minScore := 0.0
	maxScore := 5.0
	hasMin := false
	hasMax := false
	seenScores := make(map[float64]bool)
	seenLabels := make(map[string]bool)
	prevScore := -1.0
	for _, opt := range options {
		if !allowed[opt.Score] {
			return errors.New("分值仅允许 0、0.5、1、2、3、4、5")
		}
		if seenScores[opt.Score] {
			return errors.New("选项分值不能重复")
		}
		label := strings.TrimSpace(opt.Label)
		if label == "" {
			return errors.New("选项标签不能为空")
		}
		if seenLabels[label] {
			return errors.New("选项标签不能重复")
		}
		if opt.Score <= prevScore {
			return errors.New("选项分值必须从小到大排列")
		}
		seenScores[opt.Score] = true
		seenLabels[label] = true
		prevScore = opt.Score
		if opt.Score == minScore {
			hasMin = true
		}
		if opt.Score == maxScore {
			hasMax = true
		}
	}
	if !hasMin || !hasMax {
		return errors.New("必须包含最小值 0 与最大值 5")
	}
	return nil
}

func ValidatePassFailOptions(options []models.ScoreOption) error {
	if len(options) == 0 {
		return errors.New("评分选项不能为空")
	}
	seenLabels := make(map[string]bool)
	for _, opt := range options {
		if opt.Score <= 0 {
			return errors.New("选项序号必须为正整数")
		}
		label := strings.TrimSpace(opt.Label)
		if label == "" {
			return errors.New("选项不能为空")
		}
		if seenLabels[label] {
			return errors.New("选项不能重复")
		}
		seenLabels[label] = true
	}
	return nil
}

func GetActiveTemplates() ([]models.InspectionTemplate, error) {
	var templates []models.InspectionTemplate
	_, err := GetOrm().QueryTable(new(models.InspectionTemplate)).
		Filter("is_active", true).
		OrderBy("-created_at").
		All(&templates)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func FindInspectionTemplate(templateID string) (*models.InspectionTemplate, error) {
	if strings.TrimSpace(templateID) == "" {
		return nil, errors.New("模板不存在")
	}
	var template models.InspectionTemplate
	err := GetOrm().QueryTable(new(models.InspectionTemplate)).
		Filter("id", templateID).
		One(&template)
	if err != nil {
		return nil, errors.New("模板不存在")
	}
	return &template, nil
}

func SaveInspectionTemplate(template *models.InspectionTemplate, items []*models.InspectionTemplateItem) error {
	o := GetOrm()
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if template.ID == "" {
		template.ID = uuid.NewString()
		template.CreatedAt = time.Now()
		template.UpdatedAt = time.Now()
		if _, err := tx.Insert(template); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		template.UpdatedAt = time.Now()
		if _, err := tx.Update(template); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.QueryTable(new(models.InspectionTemplateItem)).
			Filter("template_id", template.ID).
			Delete(); err != nil {
			tx.Rollback()
			return err
		}
	}
	now := time.Now()
	for i, item := range items {
		item.ID = uuid.NewString()
		item.TemplateID = template.ID
		item.SortOrder = i
		item.IsActive = true
		item.CreatedAt = now
		item.UpdatedAt = now
		if _, err := tx.Insert(item); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func DeleteInspectionTemplate(templateID string) error {
	o := GetOrm()
	template, err := FindInspectionTemplate(templateID)
	if err != nil {
		return err
	}
	hasInspection := o.QueryTable(new(models.Inspection)).
		Filter("template_id", template.ID).
		Exist()
	if hasInspection {
		return errors.New("该模板已被巡店记录使用，无法删除")
	}
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.QueryTable(new(models.InspectionTemplateItem)).
		Filter("template_id", template.ID).
		Delete(); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Delete(template); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func CountTemplateItems(templateID string) (int64, error) {
	return GetOrm().QueryTable(new(models.InspectionTemplateItem)).
		Filter("template_id", templateID).
		Filter("is_active", true).
		Count()
}

// SeedInspectionTemplatesBulk 批量创建 10 条检查表模板（每条带若干检查项），用于测试数据初始化
func SeedInspectionTemplatesBulk() error {
	o := GetOrm()
	templateNames := []string{
		"门店日常巡检模板", "新店开业验收模板", "季度品质稽核模板", "月度安全专项模板", "节假日促销巡检模板",
		"新品上市陈列模板", "门店员工服务模板", "VIP 到店接待模板", "清洁卫生专项模板", "消防设备专项模板",
	}
	categoryPool := []string{"形象", "卫生", "陈列", "服务", "安全", "仓储", "员工管理", "商品管理"}
	titlePool := map[string][]struct {
		title    string
		standard string
	}{
		"形象": {
			{title: "门头招牌", standard: "招牌干净明亮，夜间灯箱正常开启，无破损缺字"},
			{title: "橱窗展示", standard: "橱窗道具摆放整齐，无破损，画面整洁"},
			{title: "店门状态", standard: "玻璃门清洁无指纹贴，把手完好，开合顺畅"},
		},
		"卫生": {
			{title: "地面清洁", standard: "地面无污渍、无垃圾、无水渍"},
			{title: "货架清洁", standard: "货架层板无积尘、无蛛网、无废弃包装"},
			{title: "洗手间", standard: "洗手间无异味、地面干爽、厕纸与洗手液充足"},
		},
		"陈列": {
			{title: "商品堆头", standard: "堆头饱满，价格标签对齐，POP 完好"},
			{title: "价签规范", standard: "每个陈列位置均有对应价签，与商品一一对应"},
			{title: "临期商品", standard: "临期商品单独陈列并有明显提示标识"},
		},
		"服务": {
			{title: "员工着装", standard: "员工统一着装，工牌佩戴正确，仪容整洁"},
			{title: "接待用语", standard: "顾客进店 3 秒内问候，使用规范礼貌用语"},
			{title: "收银操作", standard: "收银唱收唱付，双手递接小票与商品"},
		},
		"安全": {
			{title: "灭火器", standard: "灭火器在有效期内，压力指示正常，摆放位置清晰"},
			{title: "应急照明", standard: "应急照明灯完好，断电测试可正常点亮"},
			{title: "消防通道", standard: "消防通道畅通，无商品或杂物堆放"},
		},
		"仓储": {
			{title: "仓库整洁", standard: "仓库货物分类码放，通道畅通，地面无垃圾"},
			{title: "库存标识", standard: "每个货位标识清晰，品名/数量一致"},
		},
		"员工管理": {
			{title: "考勤打卡", standard: "员工按规定到岗，考勤打卡记录正常"},
			{title: "岗位培训", standard: "在岗员工已完成对应岗位培训并通过考核"},
		},
		"商品管理": {
			{title: "库存深度", standard: "畅销品不缺货，滞销品库存合理"},
			{title: "补货及时", standard: "销售空缺位 30 分钟内完成补货"},
		},
	}

	for idx, name := range templateNames {
		// 避免重复：同名模板已存在则跳过
		existCount, _ := o.QueryTable(new(models.InspectionTemplate)).Filter("name", name).Count()
		if existCount > 0 {
			continue
		}
		now := time.Now()
		tpl := &models.InspectionTemplate{
			ID:          uuid.NewString(),
			Name:        name,
			Description: name + " - 自动生成测试数据",
			IsActive:    idx%3 != 0, // 部分停用
			ScoringMode: map[bool]string{true: "additive", false: "deductive"}[idx%2 == 0],
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		// 每条模板从分类池中取 3-5 个分类，每个分类 2-3 个检查项
		catCount := 3 + idx%3
		selectedCats := categoryPool[:catCount]
		entries := make([]*models.InspectionTemplateItem, 0)
		sortOrder := 0
		for _, cat := range selectedCats {
			options := titlePool[cat]
			itemCount := 2 + idx%2
			if itemCount > len(options) {
				itemCount = len(options)
			}
			for i := 0; i < itemCount; i++ {
				scoreType := "score"
				if (sortOrder+i)%3 == 0 {
					scoreType = "pass_fail"
				}
				maxScore := 5
				requirePhoto := (sortOrder+i)%2 == 0
				requireRemark := (sortOrder+i)%2 == 1
				entries = append(entries, &models.InspectionTemplateItem{
					ID:            uuid.NewString(),
					TemplateID:    tpl.ID,
					Category:      cat,
					Title:         options[i].title,
					Standard:      options[i].standard,
					ScoreType:     scoreType,
					MaxScore:      maxScore,
					ScoreOptions:  models.MarshalScoreOptions(models.DefaultScoreOptions(scoreType, maxScore)),
					RequireRemark: requireRemark,
					RequirePhoto:  requirePhoto,
					ShowRemark:    true,
					ShowPhoto:     true,
					SortOrder:     sortOrder,
					IsActive:      true,
					CreatedAt:     now,
					UpdatedAt:     now,
				})
				sortOrder++
			}
		}
		tx, err := o.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Insert(tpl); err != nil {
			tx.Rollback()
			return err
		}
		for _, e := range entries {
			if _, err := tx.Insert(e); err != nil {
				tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
