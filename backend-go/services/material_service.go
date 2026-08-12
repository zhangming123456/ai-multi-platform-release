package services

import (
	"errors"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
)

func ListMaterials(materialType, category, keyword string, page, pageSize int) ([]models.Material, int64, error) {
	o := GetOrm()
	qs := o.QueryTable(new(models.Material)).Filter("is_active", true)
	if materialType != "" {
		qs = qs.Filter("type", materialType)
	}
	if category != "" {
		qs = qs.Filter("category", category)
	}
	if keyword != "" {
		cond := orm.NewCondition().
			Or("name__icontains", keyword).
			Or("category__icontains", keyword)
		qs = qs.SetCond(cond)
	}
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}
	var materials []models.Material
	if _, err := qs.OrderBy("-created_at").Limit(pageSize, (page-1)*pageSize).All(&materials); err != nil {
		return nil, 0, err
	}
	return materials, total, nil
}

func ListMaterialCategories(materialType string) ([]string, error) {
	o := GetOrm()
	qs := o.QueryTable(new(models.Material)).Filter("is_active", true)
	if materialType != "" {
		qs = qs.Filter("type", materialType)
	}
	var materials []models.Material
	if _, err := qs.All(&materials, "category"); err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var categories []string
	for _, m := range materials {
		cat := strings.TrimSpace(m.Category)
		if cat == "" || seen[cat] {
			continue
		}
		seen[cat] = true
		categories = append(categories, cat)
	}
	return categories, nil
}

func FindMaterial(id string) (*models.Material, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("素材不存在")
	}
	var material models.Material
	err := GetOrm().QueryTable(new(models.Material)).
		Filter("id", id).
		Filter("is_active", true).
		One(&material)
	if err != nil {
		return nil, errors.New("素材不存在")
	}
	return &material, nil
}

func SaveMaterial(material *models.Material) error {
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

func DeleteMaterial(id string) error {
	material, err := FindMaterial(id)
	if err != nil {
		return err
	}
	material.IsActive = false
	material.UpdatedAt = time.Now()
	_, err = GetOrm().Update(material)
	return err
}
