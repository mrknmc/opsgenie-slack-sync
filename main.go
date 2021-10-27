package sync

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const scheduleName string = "Engineering_schedule"
const userGroup string = "eng-support"

type Syncer struct {
	Slack    *slack.Client
	OpsGenie *schedule.Client
}

func (s *Syncer) WhoIsOnCall() ([]string, error) {
	flat := false
	scheduleResult, err := s.OpsGenie.GetOnCalls(context.TODO(), &schedule.GetOnCallsRequest{
		Flat:                   &flat,
		ScheduleIdentifierType: schedule.Name,
		ScheduleIdentifier:     scheduleName,
	})
	if err != nil {
		return nil, err
	}
	participants := scheduleResult.OnCallParticipants
	users := make([]string, len(participants))
	for i, p := range participants {
		users[i] = p.Name
	}
	return users, nil
}

func (s *Syncer) UpdateUserGroup(groupName string, ids []string) error {
	_, err := s.Slack.UpdateUserGroupMembers(userGroup, strings.Join(ids, ","))
	return err
}

func (s *Syncer) GetSlackID(email string) (string, error) {
	user, err := s.Slack.GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("<@%s>", user.ID), nil
}

func Main() {
	log.SetLevel(log.InfoLevel)
	var scheduleClient, _ = schedule.NewClient(&client.Config{
		ApiKey: os.Getenv("OPSGENIE_API_KEY"),
	})

	var syncer = Syncer{
		Slack:    slack.New(os.Getenv("SLACK_BOT_TOKEN")),
		OpsGenie: scheduleClient,
	}

	onCallUsers, err := syncer.WhoIsOnCall()
	if err != nil {
		panic(err)
	}

	// convert emails to slack ids
	slackIds := make([]string, len(onCallUsers))
	for i, email := range onCallUsers {
		slackID, err := syncer.GetSlackID(email)
		if err != nil {
			slackIds[i] = slackID
		} else {
			log.Error("Could not convert email %s to slack id", email)
		}
	}

	syncer.UpdateUserGroup(userGroup, slackIds)
}
