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
	"strconv"
	"sync"
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
	Check(ctx context.Context, place *domain.Place) (*domain.Elecprice, error)
	SetStandard(ctx context.Context, cfg *domain.ElecpriceConfig) error
	GetTobePushMSG(ctx context.Context) ([]*domain.ElectricMSG, error)
}

type elecpriceService struct {
	elecpriceDAO dao.ElecpriceDAO
	l            logger.Logger
}

func NewElecpriceService(elecpriceDAO dao.ElecpriceDAO, l logger.Logger) ElecpriceService {
	return &elecpriceService{elecpriceDAO: elecpriceDAO, l: l}
}

// Check 实现 gRPC 的 Check 方法，接收请求体并返回响应体
func (s *elecpriceService) Check(ctx context.Context, place *domain.Place) (*domain.Elecprice, error) {
	price, err := s.fetchElecPrice(ctx, place)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	return price, nil
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
			Area:      cfg.Place.Area,
			Building:  cfg.Place.Building,
			Room:      cfg.Place.Room,
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
				elecPrice, err := s.fetchElecPrice(ctx, &domain.Place{
					Area:     cfg.Area,
					Building: cfg.Building,
					Room:     cfg.Room,
				})
				if err != nil {
					errChan <- err
					return
				}

				// 转换电费数据为浮点数
				lightingRemain, err1 := strconv.ParseFloat(elecPrice.LightingRemainMoney, 64)
				airRemain, err2 := strconv.ParseFloat(elecPrice.AirRemainMoney, 64)

				// 跳过解析失败的数据
				if err1 != nil || err2 != nil {
					errChan <- fmt.Errorf("解析电费数据失败: %v, %v", err1, err2)
					return
				}

				// 检查是否符合用户设定的阈值
				if lightingRemain < float64(cfg.Money) || airRemain < float64(cfg.Money) {
					msg := &domain.ElectricMSG{
						LightingRemainMoney: &elecPrice.LightingRemainMoney,
						AirRemainMoney:      &elecPrice.AirRemainMoney,
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

func (s *elecpriceService) fetchElecPrice(ctx context.Context, place *domain.Place) (*domain.Elecprice, error) {
	for area, areaCode := range ConstantMap {
		if place.Area == area {
			var (
				LightingYesterdayUseValue string
				LightingRemainMoney       string
				LightingYesterdayUseMoney string
				AirYesterdayUseValue      string
				AirRemainMoney            string
				AirYesterdayUseMoney      string
				err                       error
			)

			wg := sync.WaitGroup{}
			errChan := make(chan error, 2) // 错误通道，用于捕获 goroutine 中的错误

			// 定义爬取函数，减少重复代码
			crawl := func(crawlFunc func(context.Context, string, string, string) (string, string, string, error),
				elecprice *string, price *string, rest *string) {
				defer wg.Done()
				var newErr error
				*elecprice, *price, *rest, newErr = crawlFunc(ctx, areaCode, place.Building, place.Room)
				if newErr != nil {
					errChan <- newErr
				}
			}

			// 启动爬取 Lighting 数据的 goroutine
			wg.Add(1)
			go crawl(CrawlLighting, &LightingRemainMoney, &LightingYesterdayUseMoney, &LightingYesterdayUseValue)

			// 启动爬取 Air 数据的 goroutine
			wg.Add(1)
			go crawl(CrawlAirCondition, &AirRemainMoney, &AirYesterdayUseValue, &AirYesterdayUseMoney)

			// 等待所有 goroutine 完成
			wg.Wait()
			close(errChan)

			// 检查错误通道是否有错误
			for e := range errChan {
				if e != nil {
					err = e
					break
				}
			}

			if err != nil {
				return nil, err
			}

			return &domain.Elecprice{
				LightingYesterdayUseValue: LightingYesterdayUseMoney,
				LightingRemainMoney:       LightingRemainMoney,
				LightingYesterdayUseMoney: LightingYesterdayUseValue,
				AirYesterdayUseValue:      AirYesterdayUseValue,
				AirRemainMoney:            AirRemainMoney,
				AirYesterdayUseMoney:      AirYesterdayUseMoney,
			}, nil

		}
	}

	return nil, errors.New("不存在的房间")
}
