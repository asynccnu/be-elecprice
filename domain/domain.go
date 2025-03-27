package domain

type Elecprice struct {
	Airconditioner *Prices `json:"airconditioner"`
	Lighting       *Prices `json:"lighting"`
}

type IDs struct {
	LightID string
	AirID   string
}

type ElecpriceConfig struct {
	Money     int64
	StudentId string // 学号
	IDs       IDs
}

type ElectricMSG struct {
	LightingRemainMoney *string
	AirRemainMoney      *string
	StudentId           string // 学号
}

type Architecture struct {
	AID  string
	Name string
}

type RoomInfo struct {
	RID  string
	Name string
}

type Prices struct {
	RemainMoney       string
	YesterdayUseValue string
	YesterdayUseMoney string
}
