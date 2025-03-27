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

func (s *ElecpriceServiceServer) GetAIDandName(ctx context.Context, req *v1.GetAIDandNameRequest) (*v1.GetAIDandNameResponse, error) {
	res, err := s.ser.GetAIDandName(ctx, req.AreaName)
	if err != nil {
		return nil, err
	}

	var resp v1.GetAIDandNameResponse
	for k, v := range res {
		resp.ArchitectureList = append(resp.ArchitectureList, &v1.GetAIDandNameResponse_Architecture{
			ArchitectureName: v,
			ArchitectureID:   k,
		})
	}
	return &resp, nil
}

func (s *ElecpriceServiceServer) GetRoomInfo(ctx context.Context, req *v1.GetRoomInfoRequest) (*v1.GetRoomInfoResponse, error) {
	res, err := s.ser.GetRoomInfo(ctx, req.ArchitectureID, req.Floor)
	if err != nil {
		return nil, err
	}

	var resp v1.GetRoomInfoResponse
	for k, v := range res {
		resp.RoomList = append(resp.RoomList, &v1.GetRoomInfoResponse_Room{
			RoomID:   k,
			RoomName: v,
		})
	}
	return &resp, nil
}

func (s *ElecpriceServiceServer) GetPrice(ctx context.Context, req *v1.GetPriceRequest) (*v1.GetPriceResponse, error) {
	res, err := s.ser.GetPrice(ctx, req.RoomAircID, req.RoomLightID)
	if err != nil {
		return nil, err
	}

	return &v1.GetPriceResponse{
		Price: &v1.GetPriceResponse_Price{
			LightingRemainMoney:       res.Lighting.RemainMoney,
			LightingYesterdayUseMoney: res.Lighting.YesterdayUseMoney,
			LightingYesterdayUseValue: res.Lighting.YesterdayUseValue,

			AirRemainMoney:       res.Airconditioner.RemainMoney,
			AirYesterdayUseMoney: res.Airconditioner.YesterdayUseMoney,
			AirYesterdayUseValue: res.Airconditioner.YesterdayUseValue,
		},
	}, nil
}

func (s *ElecpriceServiceServer) SetStandard(ctx context.Context, req *v1.SetStandardRequest) (*v1.SetStandardResponse, error) {
	err := s.ser.SetStandard(ctx, &domain.ElecpriceConfig{
		Money:     req.Money,
		StudentId: req.StudentId,
		IDs: domain.IDs{
			LightID: req.Ids.RoomLightID,
			AirID:   req.Ids.RoomAircID,
		},
	})

	return &v1.SetStandardResponse{}, err
}
