package events

import (
	"database/sql/driver"

	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// AccountsSubjectName is a subject name in NATS for the Accounts service events.
	AccountsSubjectName = "accounts"
	// SchedulerSubjectName is a subject name in NATS for the Scheduler service events.
	SchedulerSubjectName = "scheduler"
)


// CommitAndPublish commits the DB transaction and sends the given events with confirmation.
// It also handles and wraps all the errors if there is any.
func CommitAndPublish(tx driver.Tx, producer Producer, events ...*events_proto.Event) error {
	if err := utils.HandleDBError(tx.Commit()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	if len(events) > 0 {
		if err := producer.Send(events...); err != nil {
			return errors.Wrap(err, "failed to publish events")
		}
	}
	return nil
}
