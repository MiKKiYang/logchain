package grpc

import (
	"context"
	"fmt"
	"log"

	// Import generated proto code and service layer
	core "tlng/ingestion/service/core"
	pb "tlng/proto/logingestion"

	"google.golang.org/protobuf/types/known/timestamppb" // For Protobuf Timestamp
)

// Server implements the LogIngestionServer interface
type Server struct {
	pb.UnimplementedLogIngestionServer // Embed unimplemented service for forward compatibility
	svc                                *core.Service
	logger                             *log.Logger
}

// NewServer creates a new gRPC Server instance
func NewServer(s *core.Service, l *log.Logger) *Server {
	return &Server{svc: s, logger: l}
}

// SubmitLog implements the SubmitLog method in the gRPC interface
func (s *Server) SubmitLog(ctx context.Context, req *pb.SubmitLogRequest) (*pb.SubmitLogResponse, error) {
	s.logger.Println("gRPC Server: Received SubmitLog request")

	// 1. Convert Protobuf request to Service layer input structure
	input := &core.LogInput{
		LogContent:        req.GetLogContent(),
		ClientLogHash:     req.GetClientLogHash(),
		ClientSourceOrgID: req.GetClientSourceOrgId(),
	}
	// Handle optional timestamp
	if req.ClientTimestamp != nil && req.ClientTimestamp.IsValid() {
		ts := req.ClientTimestamp.AsTime()
		input.ClientTimestamp = &ts
	}

	// 2. Call core Service layer processing logic
	result, err := s.svc.SubmitLog(ctx, input)
	if err != nil {
		s.logger.Printf("gRPC Server: Service layer error: %v", err)
		// Can return different gRPC error codes based on error type
		return nil, fmt.Errorf("failed to process log submission: %w", err) // Return generic error
	}

	// 3. Convert Service layer result to Protobuf response
	response := &pb.SubmitLogResponse{
		RequestId:               result.RequestID,
		ServerLogHash:           result.ServerLogHash,
		ServerReceivedTimestamp: timestamppb.New(result.ServerReceivedTimestamp),
		Status:                  "ACCEPTED",
	}

	s.logger.Printf("gRPC Server: Successfully processed request_id: %s", result.RequestID)
	return response, nil
}

// Ensure Server implements the interface (compile-time check)
var _ pb.LogIngestionServer = (*Server)(nil)
