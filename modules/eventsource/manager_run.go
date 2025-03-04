// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package eventsource

import (
	"context"
	"time"

	"code.gitea.io/gitea/models"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/convert"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
)

// Init starts this eventsource
func (m *Manager) Init() {
	if setting.UI.Notification.EventSourceUpdateTime <= 0 {
		return
	}
	go graceful.GetManager().RunWithShutdownContext(m.Run)
}

// Run runs the manager within a provided context
func (m *Manager) Run(ctx context.Context) {
	ctx, _, finished := process.GetManager().AddTypedContext(ctx, "Service: EventSource", process.SystemProcessType, true)
	defer finished()

	then := timeutil.TimeStampNow().Add(-2)
	timer := time.NewTicker(setting.UI.Notification.EventSourceUpdateTime)
loop:
	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			break loop
		case <-timer.C:
			m.mutex.Lock()
			connectionCount := len(m.messengers)
			if connectionCount == 0 {
				log.Trace("Event source has no listeners")
				// empty the connection channel
				select {
				case <-m.connection:
				default:
				}
			}
			m.mutex.Unlock()
			if connectionCount == 0 {
				// No listeners so the source can be paused
				log.Trace("Pausing the eventsource")
				select {
				case <-ctx.Done():
					break loop
				case <-m.connection:
					log.Trace("Connection detected - restarting the eventsource")
					// OK we're back so lets reset the timer and start again
					// We won't change the "then" time because there could be concurrency issues
					select {
					case <-timer.C:
					default:
					}
					continue
				}
			}

			now := timeutil.TimeStampNow().Add(-2)

			uidCounts, err := models.GetUIDsAndNotificationCounts(then, now)
			if err != nil {
				log.Error("Unable to get UIDcounts: %v", err)
			}
			for _, uidCount := range uidCounts {
				m.SendMessage(uidCount.UserID, &Event{
					Name: "notification-count",
					Data: uidCount,
				})
			}
			then = now

			if setting.Service.EnableTimetracking {
				usersStopwatches, err := issues_model.GetUIDsAndStopwatch()
				if err != nil {
					log.Error("Unable to get GetUIDsAndStopwatch: %v", err)
					return
				}

				for _, userStopwatches := range usersStopwatches {
					apiSWs, err := convert.ToStopWatches(userStopwatches.StopWatches)
					if err != nil {
						log.Error("Unable to APIFormat stopwatches: %v", err)
						continue
					}
					dataBs, err := json.Marshal(apiSWs)
					if err != nil {
						log.Error("Unable to marshal stopwatches: %v", err)
						continue
					}
					m.SendMessage(userStopwatches.UserID, &Event{
						Name: "stopwatches",
						Data: string(dataBs),
					})
				}
			}
		}
	}
	m.UnregisterAll()
}
