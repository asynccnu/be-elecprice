//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/be-elecprice/cron"
	"github.com/asynccnu/be-elecprice/grpc"
	"github.com/asynccnu/be-elecprice/ioc"
	"github.com/asynccnu/be-elecprice/repository/dao"
	"github.com/asynccnu/be-elecprice/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		grpc.NewElecpriceGrpcService,
		service.NewElecpriceService,
		dao.NewElecpriceDAO,
		// 第三方
		ioc.InitEtcdClient,
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitGRPCxKratosServer,
		ioc.InitFeedClient,
		cron.NewElecpriceController,
		cron.NewCron,
		NewApp,
	)
	return App{}
}
