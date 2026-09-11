package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"ai-multi-platform-release/backend-go/services"
)

type HolidaysController struct {
	BaseController
}

func (c *HolidaysController) List() {
	cal, err := services.GetHolidayCalendar()
	if err != nil {
		c.WriteError(http.StatusBadGateway, "获取节假日数据失败："+err.Error())
		return
	}

	yearStr := strings.TrimSpace(c.GetQuery("year"))
	if yearStr == "" {
		c.OK(cal)
		return
	}

	year, convErr := strconv.Atoi(yearStr)
	if convErr != nil || year < 1900 || year > 2200 {
		c.WriteError(http.StatusBadRequest, "year 参数不合法")
		return
	}
	c.OK(services.FilterHolidayByYear(cal, year))
}
