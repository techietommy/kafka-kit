package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	pb "github.com/DataDog/kafka-kit/registry/protos"
)

var (
	// Errors.
	ErrFetchingTopics = errors.New("Error fetching topics")
	// Misc.
	tregex = regexp.MustCompile(".*")
)

type TopicSet map[string]*pb.Topic

// GetTopics gets topics.
func (s *Server) GetTopics(ctx context.Context, req *pb.TopicRequest) (*pb.TopicResponse, error) {
	if err := s.ValidateRequest(ctx, req, ReadRequest); err != nil {
		return nil, err
	}

	// Get topics.
	topics, err := s.fetchTopicSet(req)
	if err != nil {
		return nil, err
	}

	// Populate the response Topics field.
	resp := &pb.TopicResponse{Topics: topics}

	return resp, nil
}

// ListTopics gets topic names.
func (s *Server) ListTopics(ctx context.Context, req *pb.TopicRequest) (*pb.TopicResponse, error) {
	if err := s.ValidateRequest(ctx, req, ReadRequest); err != nil {
		return nil, err
	}

	// Get topics.
	topics, err := s.fetchTopicSet(req)
	if err != nil {
		return nil, err
	}

	// Populate the response Names field.
	resp := &pb.TopicResponse{Names: topics.Names()}

	return resp, nil
}

// fetchBrokerSet fetches metadata for all topics. If the input *pb.TopicRequest
// Name field is non-nil, the specified topic is matched if it exists. Otherwise,
// all topics found in ZooKeeper are matched. Matched topics are then filtered
// by all tags specified, if specified, in the *pb.TopicRequest tag field.
func (s *Server) fetchTopicSet(req *pb.TopicRequest) (TopicSet, error) {
	topicRegex := []*regexp.Regexp{}

	// Check if a specific topic is being fetched.
	if req.Name != "" {
		r := regexp.MustCompile(fmt.Sprintf("^%s$", req.Name))
		topicRegex = append(topicRegex, r)
	} else {
		topicRegex = append(topicRegex, tregex)
	}

	// Fetch topics from ZK.
	topics, errs := s.ZK.GetTopics(topicRegex)
	if errs != nil {
		return nil, ErrFetchingTopics
	}

	matched := TopicSet{}

	// Populate all topics.
	for _, t := range topics {
		s, _ := s.ZK.GetTopicState(t)
		matched[t] = &pb.Topic{
			Name:       t,
			Partitions: uint32(len(s.Partitions)),
			// TODO more sophisticated check than the
			// first partition len.
			Replication: uint32(len(s.Partitions["0"])),
		}
	}

	filtered, err := s.Tags.FilterTopics(matched, req.Tag)
	if err != nil {
		return nil, err
	}

	return filtered, nil
}

// Names returns a []string of topic names from a TopicSet.
func (t TopicSet) Names() []string {
	var names []string
	for n := range t {
		names = append(names, n)
	}

	return names
}
