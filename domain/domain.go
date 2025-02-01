package domain

type Elecprice struct {
	LightingRemainMoney       string // 剩余照明电费
	LightingYesterdayUseValue string // 昨日花费电量
	LightingYesterdayUseMoney string // 昨日花费电费
	AirRemainMoney            string // 空调价格
	AirYesterdayUseValue      string // 剩余照明电费
	AirYesterdayUseMoney      string // 空调剩余
}

type Place struct {
	Area     string // 区域
	Building string // 建筑
	Room     string // 房间号
}

type ElecpriceConfig struct {
	Money     int64
	StudentId string // 学号
	Place     Place
}

type ElectricMSG struct {
	LightingRemainMoney *string
	AirRemainMoney      *string
	StudentId           string // 学号
}
