package main

import (
	"context"
	"os"
	"strings"

	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const scheduleName string = "Engineering_schedule"

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
	_, err := s.Slack.UpdateUserGroupMembers(groupName, strings.Join(ids, ","))
	return err
}

func (s *Syncer) GetSlackID(email string) (string, error) {
	user, err := s.Slack.GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func main() {
	log.SetLevel(log.InfoLevel)
	scheduleClient, _ := schedule.NewClient(&client.Config{
		ApiKey: os.Getenv("OPSGENIE_API_KEY"),
	})

	syncer := Syncer{
		Slack:    slack.New(os.Getenv("SLACK_BOT_TOKEN")),
		OpsGenie: scheduleClient,
	}

	onCall, err := syncer.WhoIsOnCall()
	if err != nil {
		panic(err)
	}

	log.Debugf("People on call right now: %s", strings.Join(onCall, ","))

	// convert emails to slack ids
	slackIds := make([]string, len(onCall))
	for i, email := range onCall {
		slackID, err := syncer.GetSlackID(email)
		if err == nil {
			slackIds[i] = slackID
			log.Debugf("Slack id for email %s is %s", email, slackID)
		} else {
			log.Errorf("Could not convert email %s to slack id: %s", email, err)
		}
	}

	userGroup := os.Getenv("SLACK_USER_GROUP")
	err = syncer.UpdateUserGroup(userGroup, slackIds)
	if err == nil {
		log.Debugf("Changed user group %s to contain %s", userGroup, strings.Join(onCall, ","))
	} else {
		log.Errorf("Could not update user group %s: %s", userGroup, err)
	}
}
