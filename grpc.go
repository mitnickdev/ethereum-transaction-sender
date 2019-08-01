package main

import (
	"context"
	"database/sql"
	"fmt"
	pb "git.ddex.io/infrastructure/ethereum-launcher/messages"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

//go:generate protoc -I.  --go_out=plugins=grpc:. ./messages/messages.proto

type server struct{}

func (*server) Create(ctx context.Context, msg *pb.CreateMessage) (*pb.CreateReply, error) {
	var err error
	var value decimal.Decimal

	if msg.Value == "" {
		value = decimal.Zero
	} else {
		value, err = decimal.NewFromString(msg.Value)

		if err != nil {
			return nil, fmt.Errorf("convert value to decimal failed")
		}
	}

	var gasPrice decimal.Decimal

	if msg.GasPrice == "" {
		gasPrice = getCurrentGasPrice()
	} else {
		gasPrice, err = decimal.NewFromString(msg.GasPrice)
		if err != nil {
			return nil, fmt.Errorf("convert gas price to decimal failed")
		}
	}

	if msg.From[:2] != "0x" || len(msg.From) != 42 {
		return nil, fmt.Errorf("`form` format error, not a valid ethereum address")
	}

	if msg.To[:2] != "0x" || len(msg.To) != 42 {
		return nil, fmt.Errorf("`to` format error, not a valid ethereum address")
	}

	log := &LaunchLog{
		Hash: sql.NullString{
			Valid: false,
		},
		From:     msg.From,
		To:       msg.To,
		Value:    value,
		GasPrice: gasPrice,
		Nonce:    sql.NullInt64{},
		Data:     msg.Data,
		ItemID:   msg.ItemId,
		ItemType: msg.ItemType,
		Status:   pb.LaunchLogStatus_name[int32(pb.LaunchLogStatus_CREATED)],
	}

	if err = db.Create(log).Error; err != nil {
		return nil, err
	}

	return &pb.CreateReply{
		Status: pb.RequestStatus_REQUEST_SUCCESSFUL,
		ErrMsg: "",
	}, nil
}

func (*server) Hello(ctx context.Context, msg *pb.HelloMessage) (*pb.HelloReply, error) {
	return &pb.HelloReply{}, nil
}

func (*server) Get(ctx context.Context, msg *pb.GetMessage) (*pb.GetReply, error) {
	var logs []*LaunchLog

	if msg.Hash != "" {
		db.Where("hash = ?", msg.Hash).Find(&logs)
	} else if msg.ItemType != "" && msg.ItemId != "" {
		db.Where("item_type = ? and item_id = ?", msg.ItemType, msg.ItemId).Find(&logs)
	} else {
		return nil, fmt.Errorf("Need hash or (item_type, item_id) msg: %v", msg)
	}

	var dataLogs []*pb.Log

	for _, l := range logs {
		dataLogs = append(dataLogs, &pb.Log{
			Hash:     l.Hash.String,
			ItemId:   l.ItemID,
			ItemType: l.ItemType,
			Status:   pb.LaunchLogStatus(pb.LaunchLogStatus_value[l.Status]),
			GasPrice: l.GasPrice.String(),
		})
	}

	return &pb.GetReply{
		Status: pb.RequestStatus_REQUEST_SUCCESSFUL,
		Data:   dataLogs,
	}, nil
}

func (*server) Notify(ctx context.Context, msg *pb.NotifyMessage) (*pb.NotifyReply, error) {
	var log LaunchLog

	if msg.Hash == "" {
		return nil, fmt.Errorf("need hash")
	}

	db.Where("hash = ?", msg.Hash).First(&log)

	if log.From == "" && log.ID == 0 {
		return nil, fmt.Errorf("no such log")
	}

	err := handleLaunchLogStatus(&log, msg.Status)

	if err != nil {
		return nil, err
	}

	return &pb.NotifyReply{}, nil
}

func (*server) Subscribe(subscribeServer pb.Launcher_SubscribeServer) error {
	// TODO

	return nil
}

func startGrpcServer(ctx context.Context) {
	lis, err := net.Listen("tcp", ":3000")

	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterLauncherServer(s, &server{})

	logrus.Info("gRPC endpoint is listening on 0.0.0.0:3000\n")

	if err := s.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}
}