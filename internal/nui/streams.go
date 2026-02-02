package nui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-nui/nui/internal/ws"
)

func (a *App) HandleIndexStreams(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	listener := js.ListStreams(c.Context())
	infos := make([]*jetstream.StreamInfo, 0)
	for {
		select {
		case info, ok := <-listener.Info():
			err := listener.Err()
			if err != nil {
				if !errors.Is(err, jetstream.ErrEndOfData) {
					return a.logAndFiberError(c, err, 500)
				}
				return c.JSON(infos)
			}
			if !ok {
				return c.JSON(infos)
			}
			infos = append(infos, info)
		}
	}
}

func (a *App) HandleShowStream(c *fiber.Ctx) error {
	streamName := c.Params("stream_name")
	if streamName == "" {
		return c.Status(422).JSON("stream_name is required")
	}
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	stream, err := js.Stream(c.Context(), streamName)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	info, err := stream.Info(c.Context(), jetstream.WithSubjectFilter(">"))
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	return c.JSON(info)
}

func (a *App) HandleCreateStream(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	cfg := jetstream.StreamConfig{}
	err = c.BodyParser(&cfg)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	stream, err := js.CreateStream(c.Context(), cfg)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	info, err := stream.Info(c.Context())
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	return c.JSON(info)
}

func (a *App) HandleUpdateStream(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	cfg := jetstream.StreamConfig{}
	err = c.BodyParser(&cfg)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	stream, err := js.UpdateStream(c.Context(), cfg)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	info, err := stream.Info(c.Context())
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	return c.JSON(info)
}

func (a *App) HandleDeleteStream(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	streamName := c.Params("stream_name")
	if streamName == "" {
		return c.Status(422).JSON("stream_name is required")
	}
	_, err = js.Stream(c.Context(), streamName)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	err = js.DeleteStream(c.Context(), streamName)
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	return c.SendStatus(200)
}

func (a *App) HandlePurgeStream(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	streamName := c.Params("stream_name")
	if streamName == "" {
		return c.Status(422).JSON("stream_name is required")
	}
	stream, err := js.Stream(c.Context(), streamName)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	var options []jetstream.StreamPurgeOpt
	reqOptions := &map[string]any{}

	err = c.BodyParser(&reqOptions)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}

	if reqOptions != nil {
		for key, value := range *reqOptions {
			switch key {
			case "seq":
				if val, ok := value.(float64); ok && val >= 1 {
					options = append(options, jetstream.WithPurgeSequence(uint64(val)))
				}
			case "keep":
				if val, ok := value.(float64); ok && val >= 1 {
					options = append(options, jetstream.WithPurgeKeep(uint64(val)))
				}
			case "subject":
				if val, ok := value.(string); ok && val != "" {
					options = append(options, jetstream.WithPurgeSubject(val))
				}
			}
		}
	}
	err = stream.Purge(c.Context(), options...)
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	return c.SendStatus(204)
}

func (a *App) HandleSealStream(c *fiber.Ctx) error {
	//conn, err := a.nui.ConnPool.Get(c.Params("connection_id"))
	//if err != nil {
	//	return a.logAndFiberError(c,err,404)
	//}
	//js, err := jetstream.New(conn.Conn)
	//if err != nil {
	//	return a.logAndFiberError(c,err,422)
	//}
	//streamName := c.Params("stream_name")
	//if streamName == "" {
	//	return c.Status(422).JSON("stream_name is required")
	//}
	//stream, err := js.Stream(c.Context(), streamName)
	//if err != nil {
	//	return a.logAndFiberError(c,err,422)
	//}
	//
	//if err != nil {
	//	return a.logAndFiberError(c,err,500)
	//}
	return c.SendStatus(200)
}

func (a *App) HandleIndexStreamMessages(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	streamName := c.Params("stream_name")
	if streamName == "" {
		return c.Status(422).JSON("stream_name is required")
	}
	stream, err := js.Stream(c.Context(), streamName)
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}

	info, err := stream.Info(c.Context())
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}

	// If there are no messages in the stream, return an empty array without further processing
	if info.State.Msgs == 0 {
		return c.JSON([]ws.NatsMsg{})
	}

	// Stream has messages, ,so proceed with consumer configuration based on query parameters
	config := jetstream.ConsumerConfig{
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
		MemoryStorage: true,
		Name:          "nui-" + uuid.NewString(),
	}

	subjects := strings.Split(c.Query("subjects"), ",")
	if len(subjects) > 0 && subjects[0] != "" {
		if len(subjects) == 1 {
			config.FilterSubject = subjects[0]
		} else {
			config.FilterSubjects = subjects
		}
	}
	interval, err := strconv.Atoi(c.Query("interval"))
	if err != nil {
		interval = 25
	}
	// batch is the absolute value of the interval
	batch := interval
	if batch < 0 {
		batch = -batch
	}

	var msgCount int
	msgs := make([]ws.NatsMsg, 0, batch)

	timeStr := c.Query("start_time")
	if timeStr != "" {
		config.DeliverPolicy = jetstream.DeliverByStartTimePolicy
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return a.logAndFiberError(c, err, 422)
		}
		config.OptStartTime = &t
		msgCount = batch
	} else {
		querySeq, err := strconv.Atoi(c.Query("seq_start"))
		if err != nil {
			info, err := stream.Info(c.Context())
			if err != nil {
				return a.logAndFiberError(c, err, 500)
			}
			if interval > 0 {
				querySeq = int(info.State.FirstSeq)
			} else {
				querySeq = int(info.State.LastSeq)
			}
			querySeq = max(querySeq, 1)
		}
		var seekFromSeq uint64
		seekFromSeq, msgCount, err = findSeekSeq(c.Context(), stream, info, config, querySeq, interval)
		if err != nil {
			return a.logAndFiberError(c, err, 500)
		}
		if msgCount == 0 {
			return c.JSON(msgs)
		}
		config.OptStartSeq = seekFromSeq
	}

	msgs, err = a.fetchMessages(c, err, stream, config, batch, msgs, msgCount)
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	return c.JSON(msgs)
}

func (a *App) fetchMessages(c *fiber.Ctx, err error, stream jetstream.Stream, config jetstream.ConsumerConfig, batch int, msgs []ws.NatsMsg, msgCount int) ([]ws.NatsMsg, error) {
	consumer, err := stream.CreateOrUpdateConsumer(c.Context(), config)
	if err != nil {
		return nil, err
	}
	msgBatch, err := consumer.FetchNoWait(batch)
	if err != nil {
		return nil, err
	}
	for msg := range msgBatch.Messages() {
		if len(msgs) == msgCount {
			break
		}
		if msgBatch.Error() != nil {
			return nil, fmt.Errorf("failed to fetch message: %w", msgBatch.Error())
		}
		metadata, err := msg.Metadata()
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, ws.NatsMsg{
			Subject:    msg.Subject(),
			SeqNum:     metadata.Sequence.Stream,
			ReceivedAt: metadata.Timestamp,
			Payload:    msg.Data(),
			Headers:    msg.Headers(),
		})
	}
	_ = stream.DeleteConsumer(c.Context(), config.Name)
	return msgs, nil
}

func (a *App) HandleDeleteStreamMessage(c *fiber.Ctx) error {
	js, ok, err := a.jsOrFail(c)
	if !ok {
		return err
	}
	streamName := c.Params("stream_name")
	if streamName == "" {
		return c.Status(422).JSON("stream_name is required")
	}
	stream, err := js.Stream(c.Context(), streamName)
	if err != nil {
		return a.logAndFiberError(c, err, 500)
	}
	seq, err := strconv.Atoi(c.Params("seq"))
	if err != nil {
		return a.logAndFiberError(c, err, 422)
	}
	if seq <= 0 {
		return c.Status(422).JSON("seq must be greater than 0")
	}
	err = stream.DeleteMsg(c.Context(), uint64(seq))
	if err != nil {
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return c.Status(404).JSON("message not found")
		}
		return a.logAndFiberError(c, err, 500)
	}
	return c.SendStatus(200)
}

func findSeekSeq(ctx context.Context, stream jetstream.Stream, info *jetstream.StreamInfo, consumerConfig jetstream.ConsumerConfig, startSeq int, interval int) (uint64, int, error) {
	if interval >= 0 {
		return uint64(startSeq), interval, nil
	}

	if uint64(startSeq) < info.State.FirstSeq {
		return info.State.FirstSeq, 0, nil
	}
	intervalMultiplier := 1
	firstSeq := startSeq
	for {
		batch := min(10000, -interval*intervalMultiplier)
		firstSeq -= batch - 1
		if firstSeq <= 1 || uint64(firstSeq) <= info.State.FirstSeq {
			return info.State.FirstSeq, int(info.State.FirstSeq - uint64(firstSeq)), nil
		}
		if uint64(firstSeq) == info.State.FirstSeq {
			return info.State.FirstSeq, 1, nil
		}
		consumerConfig.OptStartSeq = uint64(firstSeq)
		consumerConfig.HeadersOnly = true
		consumer, err := stream.CreateConsumer(ctx, consumerConfig)
		if err != nil {
			return 0, 0, err
		}
		msgBatch, err := consumer.FetchNoWait(batch)
		if err != nil {
			return 0, 0, err
		}
		neededSeq, msgsCount, done, err := findSeqInBatch(msgBatch, startSeq, batch)
		err = stream.DeleteConsumer(context.Background(), consumerConfig.Name)
		if err != nil {
			jsErr, ok := err.(jetstream.JetStreamError)
			if !ok || jsErr.APIError().Code != 404 {
				return 0, 0, nil
			}
		}
		if done {
			return neededSeq, msgsCount, nil
		}
	}
}

// SubjectInfo represents information about a subject from JetStream streams
type SubjectInfo struct {
	Subject      string `json:"subject"`
	StreamName   string `json:"streamName"`
	MessageCount uint64 `json:"messageCount,omitempty"`
	Type         string `json:"type"` // "configured" or "active"
}

// HandleAvailableSubjects returns all unique subjects from all JetStream streams
// or from NATS monitoring API if JetStream is not available
// GET /api/connection/:connection_id/stream/subjects
func (a *App) HandleAvailableSubjects(c *fiber.Ctx) error {
	a.l.Info("HandleAvailableSubjects called", "connection_id", c.Params("connection_id"))

	subjects := make(map[string]SubjectInfo)

	// Try JetStream first
	js, ok, _ := a.jsOrFail(c)
	if ok {
		subjects = a.getJetStreamSubjects(c, js)
	}

	// If no JetStream subjects, try monitoring API
	if len(subjects) == 0 {
		a.l.Info("No JetStream subjects, trying monitoring API")
		monitoringSubjects := a.getMonitoringSubjects(c)
		for k, v := range monitoringSubjects {
			subjects[k] = v
		}
	}

	a.l.Info("Returning subjects", "count", len(subjects))
	return c.JSON(subjectsMapToSlice(subjects))
}

// getJetStreamSubjects fetches subjects from JetStream streams
func (a *App) getJetStreamSubjects(c *fiber.Ctx, js jetstream.JetStream) map[string]SubjectInfo {
	subjects := make(map[string]SubjectInfo)

	// First, get stream names
	streamNames := make([]string, 0)
	listener := js.ListStreams(c.Context())
	for {
		select {
		case info, ok := <-listener.Info():
			err := listener.Err()
			if err != nil {
				if !errors.Is(err, jetstream.ErrEndOfData) {
					a.l.Error("ListStreams error", "error", err)
					return subjects
				}
				goto processStreams
			}
			if !ok {
				goto processStreams
			}
			streamNames = append(streamNames, info.Config.Name)
		}
	}

processStreams:
	// Now fetch full info for each stream
	for _, streamName := range streamNames {
		stream, err := js.Stream(c.Context(), streamName)
		if err != nil {
			continue
		}

		info, err := stream.Info(c.Context(), jetstream.WithSubjectFilter(">"))
		if err != nil {
			continue
		}

		// Get configured subjects from stream config
		for _, subj := range info.Config.Subjects {
			if _, exists := subjects[subj]; !exists {
				subjects[subj] = SubjectInfo{
					Subject:    subj,
					StreamName: streamName,
					Type:       "configured",
				}
			}
		}

		// Get actual subjects with message counts
		if info.State.Subjects != nil {
			for subj, count := range info.State.Subjects {
				subjects[subj] = SubjectInfo{
					Subject:      subj,
					StreamName:   streamName,
					MessageCount: count,
					Type:         "active",
				}
			}
		}
	}

	return subjects
}

// getMonitoringSubjects fetches active subscriptions from NATS monitoring API
func (a *App) getMonitoringSubjects(c *fiber.Ctx) map[string]SubjectInfo {
	subjects := make(map[string]SubjectInfo)

	// Get connection config to find monitoring URL
	conn, err := a.nui.ConnRepo.GetById(c.Params("connection_id"))
	if err != nil {
		a.l.Error("Failed to get connection", "error", err)
		return subjects
	}

	// Try to get monitoring URL from metrics config
	monitoringURL := conn.Metrics.HttpSource.Url
	if monitoringURL == "" {
		// Try to derive from host - assume port 8222
		if len(conn.Hosts) > 0 {
			host := conn.Hosts[0]
			// Parse the NATS URL and replace port with 8222
			monitoringURL = deriveMonitoringURL(host)
		}
	}

	if monitoringURL == "" {
		a.l.Info("No monitoring URL available")
		return subjects
	}

	a.l.Info("Querying monitoring API", "url", monitoringURL)

	// Query /connz?subs=true to get all subscriptions
	connzURL := strings.TrimSuffix(monitoringURL, "/") + "/connz?subs=true"
	resp, err := http.Get(connzURL)
	if err != nil {
		a.l.Error("Failed to query monitoring API", "error", err)
		return subjects
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		a.l.Error("Monitoring API returned error", "status", resp.StatusCode)
		return subjects
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.l.Error("Failed to read monitoring response", "error", err)
		return subjects
	}

	// Parse the response
	var connzResp ConnzResponse
	if err := json.Unmarshal(body, &connzResp); err != nil {
		a.l.Error("Failed to parse monitoring response", "error", err)
		return subjects
	}

	// Extract unique subjects from all connections
	for _, connection := range connzResp.Connections {
		for _, sub := range connection.SubscriptionsList {
			// Filter out system subscriptions
			if isSystemSubject(sub) {
				continue
			}
			if _, exists := subjects[sub]; !exists {
				subjects[sub] = SubjectInfo{
					Subject:    sub,
					StreamName: "core",
					Type:       "subscription",
				}
			}
		}
	}

	a.l.Info("Found monitoring subjects", "count", len(subjects))
	return subjects
}

// ConnzResponse represents the NATS /connz response
type ConnzResponse struct {
	Connections []ConnzConnection `json:"connections"`
}

type ConnzConnection struct {
	SubscriptionsList []string `json:"subscriptions_list"`
}

// deriveMonitoringURL tries to derive monitoring URL from NATS connection URL
func deriveMonitoringURL(natsURL string) string {
	// Handle formats: nats://host:port, host:port, nats://user:pass@host:port
	parsed, err := url.Parse(natsURL)
	if err != nil {
		// Try as host:port
		if strings.Contains(natsURL, ":") {
			parts := strings.Split(natsURL, ":")
			if len(parts) >= 1 {
				return "http://" + parts[0] + ":8222"
			}
		}
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	return "http://" + host + ":8222"
}

// isSystemSubject returns true if the subject is a system subject that should be filtered
func isSystemSubject(subject string) bool {
	// Filter all subjects starting with $ (system subjects like $SYS, $JS, $KV, $SRV, etc.)
	if strings.HasPrefix(subject, "$") {
		return true
	}

	// Filter internal subjects
	systemPrefixes := []string{
		"_INBOX",
		"_R_",
		"_STAN",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(subject, prefix) {
			return true
		}
	}

	return false
}

func subjectsMapToSlice(subjects map[string]SubjectInfo) []SubjectInfo {
	result := make([]SubjectInfo, 0, len(subjects))
	for _, info := range subjects {
		result = append(result, info)
	}
	return result
}

func findSeqInBatch(msgBatch jetstream.MessageBatch, startSeq int, batch int) (uint64, int, bool, error) {
	neededSeq := uint64(0)
	msgsCount := 0
	for msg := range msgBatch.Messages() {
		if msgBatch.Error() != nil {
			return 0, 0, false, msgBatch.Error()
		}
		msgsCount++
		metadata, err := msg.Metadata()
		if err != nil {
			return 0, 0, false, err
		}
		if neededSeq == 0 {
			neededSeq = metadata.Sequence.Stream
		}
		if metadata.Sequence.Stream > uint64(startSeq) {
			return 0, 0, false, nil
		}
		if msgsCount >= batch {
			return neededSeq, msgsCount, true, nil
		}
	}
	return 0, 0, false, nil
}
