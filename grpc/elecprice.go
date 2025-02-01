package grpc

import (
	"context"
	v1 "github.com/asynccnu/be-api/gen/proto/elecprice/v1"
	"github.com/asynccnu/be-elecprice/domain"
	"github.com/asynccnu/be-elecprice/service"
	"google.golang.org/grpc"
)

type ElecpriceServiceServer struct {
	v1.UnimplementedElecpriceServiceServer
	ser service.ElecpriceService
}

func NewElecpriceGrpcService(ser service.ElecpriceService) *ElecpriceServiceServer {
	return &ElecpriceServiceServer{ser: ser}
}

func (s *ElecpriceServiceServer) Register(server grpc.ServiceRegistrar) {
	v1.RegisterElecpriceServiceServer(server, s)
}

func (s *ElecpriceServiceServer) Check(ctx context.Context, req *v1.CheckRequest) (*v1.CheckResponse, error) {
	elecprice, err := s.ser.Check(ctx, &domain.Place{
		Area:     req.Place.GetArea(),
		Building: req.Place.GetBuilding(),
		Room:     req.Place.GetRoom(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.CheckResponse{Price: &v1.Price{
		LightingRemainMoney:       elecprice.LightingRemainMoney,
		LightingYesterdayUseValue: elecprice.LightingYesterdayUseValue,
		LightingYesterdayUseMoney: elecprice.LightingYesterdayUseMoney,
		AirRemainMoney:            elecprice.AirRemainMoney,
		AirYesterdayUseValue:      elecprice.AirYesterdayUseValue,
		AirYesterdayUseMoney:      elecprice.AirYesterdayUseMoney,
	}}, nil
}

func (s *ElecpriceServiceServer) SetStandard(ctx context.Context, req *v1.SetStandardRequest) (*v1.SetStandardResponse, error) {
	err := s.ser.SetStandard(ctx, &domain.ElecpriceConfig{
		Money:     req.GetMoney(),
		StudentId: req.GetStudentId(),
		Place: domain.Place{
			Area:     req.Place.GetArea(),
			Building: req.Place.GetBuilding(),
			Room:     req.Place.GetRoom(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &v1.SetStandardResponse{}, nil
}
