package utils

import (
	"gotest.tools/assert"
	"os"
	"testing"
	"time"
)

func TestTimeEvent(t *testing.T) {
	output := "result.txt"
	os.Remove(output)

	event := NewTimeEvent("test logger", "result.txt")
	event1 := AddSubEvent(event, "event1")
	time.Sleep(1 * time.Second)
	EndEvent(event1)

	event2 := AddSubEvent(event, "event2")
	event2_1 := AddSubEvent(event2, "event2-1")
	time.Sleep(2 * time.Second)
	EndEvent(event2_1)
	event2_2 := AddSubEvent(event2, "event2-2")
	time.Sleep(1 * time.Second)
	EndEvent(event2_2)
	EndEvent(event2)

	event3 := AddSubEvent(event, "event3")
	time.Sleep(3 * time.Second)
	EndEvent(event3)

	err := event.Output()
	assert.NilError(t, err)
}
