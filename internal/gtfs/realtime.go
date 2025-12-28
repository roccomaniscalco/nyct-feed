package gtfs

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"nyct-feed/internal/pb"
)

var feedUrls = [8]string{
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-g",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-jz",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-si",
}

// GetRealtime fetches GTFS updates for all realtime feeds concurrently
func GetRealtime() ([]*pb.FeedMessage, error) {
	msgs := make([]*pb.FeedMessage, len(feedUrls))
	var g errgroup.Group

	for i, feedUrl := range feedUrls {
		g.Go(func() error {
			i, feedUrl := i, feedUrl // capture loop variables

			msg, err := fetchFeedMessage(feedUrl)
			if err != nil {
				return err
			}

			msgs[i] = msg
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch feeds: %v", err)
	}
	return msgs, nil
}

func fetchFeedMessage(feedUrl string) (*pb.FeedMessage, error) {
	resp, err := http.Get(feedUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	msg := &pb.FeedMessage{}
	if err := proto.Unmarshal(body, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// Write feed messages to /out. Helpful for debugging
func writeFeedMessage(msg *pb.FeedMessage) {
	marshallOptions := protojson.MarshalOptions{
		Indent: "  ",
	}

	feedJson, err := marshallOptions.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(dataDir, dirPerms)
	if err != nil {
		log.Fatal(err)
	}

	outFile := fmt.Sprintf("%smta-feed-%d.json", dataDir, *msg.Header.Timestamp)

	err = os.WriteFile(outFile, feedJson, filePerms)
	if err != nil {
		log.Fatal(err)
	}
}
