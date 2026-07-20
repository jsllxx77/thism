package models

type DashboardSettings struct {
	ShowDashboardCardIP  bool `json:"show_dashboard_card_ip"`
	ShowSystemPressure   bool `json:"show_system_pressure"`
	ShowMemoryPressure   bool `json:"show_memory_pressure"`
}
