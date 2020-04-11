package gocron_test

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron"
)

var task = func() {
	fmt.Println("I am a task")
}

// Job

func ExampleJob_GetScheduledDateTime() {
	job, _ := s.Every(1).Day().At("10:30").Do(task)

	fmt.Println(job.GetScheduledDateTime())
	// Output: 2020-04-11 10:30:00 +0000 UTC
}

func ExampleJob_GetScheduledTime() {
	job, _ := s.Every(1).Day().At("10:30").Do(task)

	fmt.Println(job.GetScheduledTime())
	// Output: 10:30
}

func ExampleJob_GetWeekday() {
	weekdayJob, _ := s.Every(1).Weekday(time.Wednesday).Do(task)

	weekday, err := weekdayJob.GetWeekday()
	fmt.Println(weekday, err)
	//Output: Wednesday <nil>
}

func ExampleJob_Tag() {
	j, _ := s.Every(1).Minute().Do(task)
	j.Tag("foo")
	j.Tag("bar")

	fmt.Println(j.Tags())
	//Output: [foo bar]
}

func ExampleJob_Tags() {
	j, _ := s.Every(1).Minute().Do(task)
	j.Tag("foo")
	j.Tag("bar")

	fmt.Println(j.Tags())
	//Output: [foo bar]
}

func ExampleJob_UnTag() {
	j, _ := s.Every(1).Minute().Do(task)
	j.Tag("foo")
	j.Tag("bar")
	j.Untag("foo")

	fmt.Println(j.Tags())
	//Output: [bar]
}

// Scheduler

var s = gocron.NewScheduler(time.UTC)

func ExampleNewScheduler() {
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Second().Do(task)
}

func ExampleScheduler_At() {
	s.Every(1).Day().At("10:30").Do(task)
	s.Every(1).Monday().At("10:30:01").Do(task)
}

func ExampleScheduler_Clear() {
	s.Every(1).Second().Do(task)

	fmt.Println(len(s.Jobs()))

	s.Clear()
	fmt.Println(len(s.Jobs()))
	//Output: 1
	//0
}

func ExampleScheduler_Day() {
	s.Every(1).Day().Do(task)
}

func ExampleScheduler_Days() {
	s.Every(2).Days().Do(task)
}

func ExampleScheduler_Do() {
	s.Every(1).Second().Do(task)
}

func ExampleScheduler_Every() {
	s.Every(1).Second().Do(task)
}

func ExampleScheduler_StartBlocking() {
	s.Every(3).Seconds().Do(task)

	s.StartBlocking()

	fmt.Println("hello, world!")
	// Output: I am a task
}

func ExampleScheduler_StartAsync() {
	s.Every(3).Seconds().Do(task)

	s.StartAsync()

	fmt.Println("hello, world!")
	// Output: I am a task
	// hello, world!
}

func ExampleScheduler_StartAt() {
	t := time.Date(2019, time.November, 10, 15, 0, 0, 0, time.UTC)

	s.Every(1).Hour().StartAt(t).Do(task)
}

func ExampleScheduler_Weekday() {
	task := func() { fmt.Println("I am a task") }

	s.Every(1).Weekday(time.Monday).Do(task)

	// Same as
	s.Every(1).Monday().Do(task)
}
