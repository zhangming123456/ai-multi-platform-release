package models

import (
	"time"
)

type Store struct {
	ID        string    `orm:"column(id);pk;size(36)" json:"id"`
	Name      string    `orm:"column(name);size(200)" json:"name"`
	Code      string    `orm:"column(code);size(50);null" json:"code"`
	Address   string    `orm:"column(address);size(500);null" json:"address"`
	Contact   string    `orm:"column(contact);size(100);null" json:"contact"`
	Phone     string    `orm:"column(phone);size(50);null" json:"phone"`
	ManagerID string    `orm:"column(manager_id);size(36);null" json:"manager_id"`
	Status    string    `orm:"column(status);size(20);default(active)" json:"status"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (s *Store) TableName() string {
	return "stores"
}
