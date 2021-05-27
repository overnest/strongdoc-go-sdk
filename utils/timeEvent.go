package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type TimeEvent struct {
	startTime   time.Time
	endTime     time.Time
	duration    int64
	description string
	children    []*TimeEvent
	filepath    string
}

func EndEvent(event *TimeEvent) {
	if event != nil {
		event.end()
	}
}

func AddSubEvent(event *TimeEvent, description string) *TimeEvent {
	if event != nil {
		return event.addSubEvent(description)
	}
	return nil
}

func (e *TimeEvent) end() {
	if !e.endTime.IsZero() { // already ended
		return
	}
	e.endTime = time.Now()
}

func (e *TimeEvent) addSubEvent(description string) *TimeEvent {
	subEvent := &TimeEvent{
		startTime:   time.Now(),
		description: description,
	}
	e.children = append(e.children, subEvent)
	return subEvent
}

func NewTimeEvent(title string, filepath string) *TimeEvent {
	return &TimeEvent{
		startTime:   time.Now(),
		description: title,
		filepath:    filepath,
	}
}

func (e *TimeEvent) Output() error {
	e.end()
	file, err := createOrOpenLocalFile(e.filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	print(nil, e, file, 0)
	return nil
}

func (e *TimeEvent) OutputSelf() error {
	e.end()
	file, err := createOrOpenLocalFile(e.filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	e.duration = e.GetDuration()
	eventLog := fmt.Sprintf("description %v\nduration %v ms\n", e.description, e.duration)
	file.WriteString(eventLog)
	fmt.Print(eventLog)
	return nil
}

func (e *TimeEvent) GetDuration() int64 {
	e.end()
	return e.endTime.Sub(e.startTime).Milliseconds()
}

// print TimeEvent and write result to file
func print(parent, child *TimeEvent, file *os.File, level int) error {
	child.duration = child.GetDuration()

	var eventLog string
	if parent == nil {
		eventLog = fmt.Sprintf("%v%v %v\n",
			multipleTabs(level),
			child.description,
			child.duration)
	} else {
		eventLog = fmt.Sprintf("%v%v %v %.2f%%\n",
			multipleTabs(level),
			child.description,
			child.duration,
			float64(child.duration)/float64(parent.duration)*100)
	}
	err := writeAndPrintString(file, eventLog)
	if err != nil {
		return err
	}
	for _, subEvent := range child.children {
		err := print(child, subEvent, file, level+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeAndPrintString(file *os.File, data string) error {
	if file != nil {
		_, err := file.WriteString(data)
		if err != nil {
			return err
		}
	}
	fmt.Print(data)
	return nil
}

func multipleTabs(n int) string {
	return strings.Repeat("\t", n)
}
