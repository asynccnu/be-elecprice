package service

import (
	"context"
	"errors"
	"fmt"
	elecpricev1 "github.com/asynccnu/be-api/gen/proto/elecprice/v1"
	"github.com/asynccnu/be-elecprice/domain"
	"github.com/asynccnu/be-elecprice/pkg/errorx"
	"github.com/asynccnu/be-elecprice/pkg/logger"
	"github.com/asynccnu/be-elecprice/repository/dao"
	"github.com/asynccnu/be-elecprice/repository/model"
	"net/url"
	"strconv"
	"sync"
	"time"
)

var (
	INTERNET_ERROR = func(err error) error {
		return errorx.New(elecpricev1.ErrorInternetError("网络错误"), "net", err)
	}
	FIND_CONFIG_ERROR = func(err error) error {
		return errorx.New(elecpricev1.ErrorFindConfigError("获取配置失败"), "dao", err)
	}
	SAVE_CONFIG_ERROR = func(err error) error {
		return errorx.New(elecpricev1.ErrorSaveConfigError("保存配置失败"), "dao", err)
	}
)

type ElecpriceService interface {
	SetStandard(ctx context.Context, cfg *domain.ElecpriceConfig) error
	GetTobePushMSG(ctx context.Context) ([]*domain.ElectricMSG, error)
	GetAIDandName(ctx context.Context, area string) (map[string]string, error)
	GetRoomInfo(ctx context.Context, archiID string, floor string) (map[string]string, error)
	GetPrice(ctx context.Context, area string, floor string) (*domain.Elecprice, error)
}

type elecpriceService struct {
	elecpriceDAO dao.ElecpriceDAO
	l            logger.Logger
}

func NewElecpriceService(elecpriceDAO dao.ElecpriceDAO, l logger.Logger) ElecpriceService {
	return &elecpriceService{elecpriceDAO: elecpriceDAO, l: l}
}

func (s *elecpriceService) SetStandard(ctx context.Context, cfg *domain.ElecpriceConfig) error {
	// 查询是否存在记录
	conf, err := s.elecpriceDAO.First(ctx, cfg.StudentId)
	if err != nil && !s.elecpriceDAO.IsNotFoundError(err) { // 非 NotFound 错误直接返回
		return INTERNET_ERROR(fmt.Errorf("查询配置失败: %w", err))
	}

	if conf == nil {
		// 构造需要保存的数据
		conf = &model.ElecpriceConfig{
			StudentID: cfg.StudentId,
			Money:     cfg.Money,
			LightID:   cfg.IDs.LightID,
			AirID:     cfg.IDs.AirID,
		}
	}

	err = s.elecpriceDAO.Save(ctx, conf)
	if err != nil {
		return err
	}

	return nil
}

func (s *elecpriceService) GetTobePushMSG(ctx context.Context) ([]*domain.ElectricMSG, error) {
	var (
		resultMsgs []*domain.ElectricMSG       // 存储最终结果
		lastID     int64                 = -1  // 初始游标为 -1，表示从头开始
		limit      int                   = 100 // 每次分页查询的大小
	)

	// 用于控制并发量的通道（令牌池），限制同时运行的 goroutine 数量为 10
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	for {
		// 分页获取配置数据
		configs, nextID, err := s.elecpriceDAO.GetConfigsByCursor(ctx, lastID, limit)
		if err != nil {
			return nil, err
		}

		// 如果没有更多数据，跳出循环
		if len(configs) == 0 {
			break
		}

		// 用于并发处理的 goroutine
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			errChan = make(chan error, len(configs))
		)

		for _, config := range configs {
			wg.Add(1)
			// 获取一个令牌（阻塞直到可用）
			semaphore <- struct{}{}

			go func(cfg model.ElecpriceConfig) {
				defer wg.Done()
				// 释放令牌
				defer func() { <-semaphore }()

				// 获取房间的实时电费
				elecPrice, err := s.GetPrice(ctx, cfg.AirID, cfg.LightID)

				if err != nil {
					errChan <- err
					return
				}

				// 转换电费数据为浮点数
				lightingRemain, err1 := strconv.ParseFloat(elecPrice.Lighting.RemainMoney, 64)
				airRemain, err2 := strconv.ParseFloat(elecPrice.Airconditioner.RemainMoney, 64)

				// 跳过解析失败的数据
				if err1 != nil || err2 != nil {
					errChan <- fmt.Errorf("解析电费数据失败: %v, %v", err1, err2)
					return
				}

				// 检查是否符合用户设定的阈值
				if lightingRemain < float64(cfg.Money) || airRemain < float64(cfg.Money) {
					msg := &domain.ElectricMSG{
						LightingRemainMoney: &elecPrice.Lighting.RemainMoney,
						AirRemainMoney:      &elecPrice.Airconditioner.RemainMoney,
						StudentId:           cfg.StudentID,
					}

					// 并发安全地添加结果
					mu.Lock()
					resultMsgs = append(resultMsgs, msg)
					mu.Unlock()
				}
			}(config)
		}

		// 等待所有 goroutine 完成
		wg.Wait()
		close(errChan)

		// 检查是否有错误
		for err := range errChan {
			if err != nil {
				// 可以选择返回第一个错误，或者记录日志
				return nil, err
			}
		}

		// 更新游标
		lastID = nextID
	}
	return resultMsgs, nil
}

func (s *elecpriceService) GetAIDandName(ctx context.Context, area string) (map[string]string, error) {
	for name_, code := range ConstantMap {
		if area == name_ {
			body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getArchitectureInfo?Area_ID=%s", code))
			if err != nil {
				return nil, INTERNET_ERROR(err)
			}
			rege := `<ArchitectureID>(\d+)</ArchitectureID>\s*<ArchitectureName>(.*?)</ArchitectureName>`
			res, err := matchRegex(body, rege)
			if err != nil {
				return nil, INTERNET_ERROR(err)
			}

			return res, nil

		}
	}
	return nil, errors.New("不存在的区域")
}

func (s *elecpriceService) GetRoomInfo(ctx context.Context, archiID string, floor string) (map[string]string, error) {
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomInfo?Architecture_ID=%s&Floor=%s", archiID, floor))
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	rege := `<RoomNo>(\d+)</RoomNo>\s*<RoomName>(.*?)</RoomName>`
	res, err := matchRegex(body, rege)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	return res, nil
}

func (s *elecpriceService) GetPrice(ctx context.Context, aroomid string, lroomid string) (*domain.Elecprice, error) {
	amid, err := s.GetMeterID(ctx, aroomid)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	lmid, err := s.GetMeterID(ctx, lroomid)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	airc, err := s.GetFinalInfo(ctx, amid)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	light, err := s.GetFinalInfo(ctx, lmid)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	return &domain.Elecprice{
		Airconditioner: airc,
		Lighting:       light,
	}, nil
}

func (s *elecpriceService) GetMeterID(ctx context.Context, RoomID string) (string, error) {
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomMeterInfo?Room_ID=%s", RoomID))
	if err != nil {
		return "", INTERNET_ERROR(err)
	}

	rege := `<meterId>(.*?)</meterId>`
	id, err := matchRegexpOneEle(body, rege)
	if err != nil {
		return "", INTERNET_ERROR(err)
	}

	return id, nil
}

func (s *elecpriceService) GetFinalInfo(ctx context.Context, meterID string) (*domain.Prices, error) {
	//取余额
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getReserveHKAM?AmMeter_ID=%s", meterID))
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	reg1 := `<remainPower>(.*?)</remainPower>`
	remain, err := matchRegexpOneEle(body, reg1)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	//取昨天消费
	encodedDate := url.QueryEscape(time.Now().AddDate(0, 0, -1).Format("2006/1/2"))
	body, err = sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getMeterDayValue?AmMeter_ID=%s&startDate=%s&endDate=%s", meterID, encodedDate, encodedDate))
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	reg2 := `<dayValue>(.*?)</dayValue>`
	dayValue, err := matchRegexpOneEle(body, reg2)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	reg3 := `<dayUseMeony>(.*?)</dayUseMeony>`
	dayUseMeony, err := matchRegexpOneEle(body, reg3)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	finalInfo := &domain.Prices{
		RemainMoney:       remain,
		YesterdayUseMoney: dayUseMeony,
		YesterdayUseValue: dayValue,
	}
	return finalInfo, nil
}
