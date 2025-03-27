package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// 通用 HTTP 请求函数
func sendRequest(ctx context.Context, url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	return string(body), nil
}

// 匹配正则工具
func matchRegex(input, pattern string) (map[string]string, error) {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(input, -1)
	if matches == nil {
		return nil, errors.New("未匹配到结果")
	}
	res := make(map[string]string)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		//"xx":"123"
		res[match[1]] = match[2]
	}
	return res, nil
}

func matchRegexpOneEle(input, pattern string) (string, error) {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return "", errors.New("未匹配到结果")
	}
	if len(matches) < 2 {
		return "", errors.New("未匹配到结果")
	}
	return matches[1], nil
}

//// 爬取单元号
//func getArchitectureID(ctx context.Context, areaCode, building string) (code string, err error) {
//	url := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getArchitectureInfo?Area_ID=%s", areaCode)
//	body, err := sendRequest(ctx, url)
//	if err != nil {
//		return "", fmt.Errorf("获取单元号失败: %w", err)
//	}
//
//	pattern := `<ArchitectureID>(\d+)</ArchitectureID>\s*<ArchitectureName>` + regexp.QuoteMeta(building) + `</ArchitectureName>`
//	match, err := matchRegex(body, pattern)
//	if err != nil {
//		return "", fmt.Errorf("解析单元号失败: %w", err)
//	}
//
//	return match[1], nil
//}
//
//// 爬取房间号
//func getRoomID(ctx context.Context, architectureID, room, roomType string) (string, error) {
//	// 提取房间楼层信息
//	floor := string(room[0])
//	// 构建请求 URL
//	url := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomInfo?Architecture_ID=%s&Floor=%s", architectureID, floor)
//
//	// 发送请求并获取响应
//	body, err := sendRequest(ctx, url)
//	if err != nil {
//		return "", fmt.Errorf("获取房间号失败: %w", err)
//	}
//
//	// 匹配房间号和房间名的正则表达式
//	const pattern = `<RoomNo>(\d+)</RoomNo>\s*<RoomName>(.*?)</RoomName>`
//	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(body, -1)
//
//	// 遍历匹配结果，查找符合条件的房间
//	for _, match := range matches {
//		if len(match) < 2 {
//			continue
//		}
//		roomName := match[2]
//		roomNo := match[1]
//
//		// 如果是空调类型，需要额外排除包含"A"的房间
//		if roomType == "空调" {
//			if strings.Contains(roomName, room) && strings.Contains(roomName, roomType) {
//				return roomNo, nil
//			} else if strings.Contains(roomName, room) && !strings.Contains(roomName, "照明") && !strings.Contains(roomName, "A") {
//				return roomNo, nil
//			}
//		}
//		// 如果是空调类型，需要额外排除包含"A"的房间
//		if roomType == "照明" {
//			if strings.Contains(roomName, room) && strings.Contains(roomName, roomType) {
//				return roomNo, nil
//			} else if strings.Contains(roomName, room) && !strings.Contains(roomName, "空调") && !strings.Contains(roomName, "A") {
//				return roomNo, nil
//			}
//		}
//	}
//
//	return "", errors.New("未找到匹配的房间号")
//}
//
//// 爬取MeterId
//func getMeterId(ctx context.Context, roomID string) (string, error) {
//
//	url := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomMeterInfo?Room_ID=%s", roomID)
//	pattern := `<meterId>(.*?)</meterId>`
//
//	body, err := sendRequest(ctx, url)
//	if err != nil {
//		return "", fmt.Errorf("获取最终数据失败: %w", err)
//	}
//
//	match, err := matchRegex(body, pattern)
//	if err != nil {
//		return "", fmt.Errorf("解析最终数据失败: %w", err)
//	}
//
//	return match[1], nil
//}
//
//// 爬取空调数据
//func CrawlAirCondition(ctx context.Context, areaCode, building, room string) (string, string, string, error) {
//	// 获取建筑ID
//	architectureID, err := getArchitectureID(ctx, areaCode, building)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取单元号失败: %w", err)
//	}
//
//	// 获取房间ID
//	roomID, err := getRoomID(ctx, architectureID, room, "空调")
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取房间号失败: %w", err)
//	}
//
//	// 获取计量表ID
//	meterId, err := getMeterId(ctx, roomID)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取空调数据失败: %w", err)
//	}
//
//	// 获取剩余电费
//	remainMoney, err := getRemainingPower(ctx, meterId)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取剩余电费失败: %w", err)
//	}
//
//	// 获取昨日用电量和费用
//	dayValue, dayUseMoney, err := getYesterdayUsage(ctx, meterId)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取昨日用电量和费用失败: %w", err)
//	}
//
//	// 返回数据
//	return remainMoney, dayValue, dayUseMoney, nil
//}
//
//// 爬取照明数据
//func CrawlLighting(ctx context.Context, area, building, room string) (string, string, string, error) {
//	// 获取建筑ID
//	architectureID, err := getArchitectureID(ctx, area, building)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取单元号失败: %w", err)
//	}
//
//	// 获取房间ID
//	roomID, err := getRoomID(ctx, architectureID, room, "照明")
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取房间号失败: %w", err)
//	}
//
//	// 获取计量表ID
//	meterId, err := getMeterId(ctx, roomID)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取照明数据失败: %w", err)
//	}
//
//	// 获取剩余电费
//	remainPower, err := getRemainingPower(ctx, meterId)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取剩余电费失败: %w", err)
//	}
//
//	// 获取昨日用电量和费用
//	dayValue, dayUseMoney, err := getYesterdayUsage(ctx, meterId)
//	if err != nil {
//		return "", "", "", fmt.Errorf("获取昨日用电量和费用失败: %w", err)
//	}
//
//	// 返回数据
//	return remainPower, dayValue, dayUseMoney, nil
//}
//
//// 获取剩余电费
//func getRemainingPower(ctx context.Context, meterId string) (string, error) {
//	// 创建 HTTP 客户端
//
//	url := "https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getReserveHKAM?AmMeter_ID=" + meterId
//	body, err := sendRequest(ctx, url)
//	if err != nil {
//		return "", err
//	}
//
//	re := regexp.MustCompile(`<remainPower>(.*?)</remainPower>`)
//	match := re.FindStringSubmatch(body)
//	if len(match) < 2 {
//		return "", errors.New("未匹配到剩余电费数据")
//	}
//
//	return match[1], nil
//}
//
//// 获取昨日用电量和费用
//func getYesterdayUsage(ctx context.Context, meterId string) (string, string, error) {
//
//	// 获取昨天日期
//	yesterday := time.Now().AddDate(0, 0, -1).Format("2006/1/2")
//	url := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getMeterDayValue?AmMeter_ID=%s&startDate=%s&endDate=%s", meterId, yesterday, yesterday)
//
//	body, err := sendRequest(ctx, url)
//	if err != nil {
//		return "", "", err
//	}
//
//	reValue := regexp.MustCompile(`<dayValue>(.*?)</dayValue>`)
//	reMoney := regexp.MustCompile(`<dayUseMeony>(.*?)</dayUseMeony>`)
//
//	valueMatch := reValue.FindStringSubmatch(body)
//	moneyMatch := reMoney.FindStringSubmatch(body)
//
//	if len(valueMatch) < 2 || len(moneyMatch) < 2 {
//		return "", "", errors.New("未匹配到昨日用电量或费用数据")
//	}
//
//	return valueMatch[1], moneyMatch[1], nil
//}
