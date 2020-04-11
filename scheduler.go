package gocron

import (
	"fmt"
	"reflect"
	"time"
)

type JobSlice []*Job

// Scheduler struct stores a list of Jobs and the location of time Scheduler
// Scheduler implements the sort.Interface{} for sorting Jobs, by the time of nextRun
type Scheduler struct {
	jobsAtTime map[time.Duration]JobSlice // jobs that will run at a given time
	jobs       JobSlice                   // every job associated with this scheduler
	loc        *time.Location
	time       timeHelper // an instance of timeHelper to interact with the time package
}

// NewScheduler creates a new Scheduler
func NewScheduler(loc *time.Location) *Scheduler {
	return &Scheduler{
		jobsAtTime: make(map[time.Duration]JobSlice),
		jobs:       newJobsSlice(),
		loc:        loc,
	}
}
func newJobsSlice() JobSlice {
	return make(JobSlice, 0, 0)
}

// StartBlocking starts all the pending jobs using a second-long ticker and blocks the current thread
func (s *Scheduler) StartBlocking() {
	<-s.StartAsync()
}

// StartAsync starts a goroutine that runs all the pending using a second-long ticker
func (s *Scheduler) StartAsync() chan struct{} {
	stopped := make(chan struct{})
	var tickers []*time.Ticker

	// for each schedule time, runs all jobs associated with it.
	//  Currently we start a goroutine with a select for each timeframe
	// A better approach would be to have a single select for all timeframes,
	// creating an or() function that returns at any channel response
	for duration, jobs := range s.jobsAtTime {
		ticker := time.NewTicker(duration)
		tickers = append(tickers, ticker)
		go func(jobs JobSlice) {
			for {
				select {
				case <-ticker.C:
					if len(jobs) == 0 {
						ticker.Stop()
						return
					}
					for _, job := range jobs {
						go job.run()
					}
				case <-stopped:
					ticker.Stop()
					return
				}
			}
		}(jobs)
	}

	return stopped
}

// Jobs returns the list of Jobs from the Scheduler
func (s *Scheduler) Jobs() JobSlice {
	return s.jobs
}

// Len returns the number of Jobs in the Scheduler
func (s *Scheduler) Len() int {
	return len(s.jobs)
}

// SetLocation changes the default time location
func (s *Scheduler) SetLocation(newLocation *time.Location) {
	s.loc = newLocation
}

// NextRun datetime when the next Job should run. FIXME
//func (s *Scheduler) NextRun() (*Job, time.Time) {
//	if len(s.jobs) <= 0 {
//		return nil, s.time.Now(s.loc)
//	}
//	sort.Sort(s)
//	return s.jobs[0], s.jobs[0].nextRun
//}

// Every schedules a new periodic Job with interval
func (s *Scheduler) Every(interval uint64) *Scheduler {
	job := NewJob(interval)
	s.jobs = append(s.jobs, job)
	return s
}

func (s *Scheduler) run(job *Job) error {
	if job.lock {
		if locker == nil {
			return fmt.Errorf("trying to lock %s with nil locker", job.jobFunc)
		}
		key := getFunctionKey(job.jobFunc)

		locker.Lock(key)
		defer locker.Unlock(key)
	}
	go job.run()

	return nil
}

// RunAll run all Jobs regardless if they are scheduled to run or not
func (s *Scheduler) RunAll() {
	s.RunAllWithDelay(0)
}

// RunAllWithDelay runs all Jobs with delay seconds
func (s *Scheduler) RunAllWithDelay(d int) {
	for _, job := range s.jobs {
		err := s.run(job)
		if err != nil {
			// add to err slice
		}
		s.time.Sleep(time.Duration(d) * time.Second)
	}
}

// Remove specific Job j by function
func (s *Scheduler) Remove(j interface{}) {
	s.removeByCondition(func(someJob *Job) bool {
		return someJob.jobFunc == getFunctionName(j)
	})
}

// RemoveByRef removes specific Job j by reference
func (s *Scheduler) RemoveByRef(j *Job) {
	s.removeByCondition(func(someJob *Job) bool {
		return someJob == j
	})
}

func (s *Scheduler) removeByCondition(shouldRemove func(*Job) bool) {
	// remove from jobs slice
	for i, job := range s.jobs {
		if shouldRemove(job) {
			s.jobs = removeAtIndex(s.jobs, i)
		}
	}
	// remove from time slice
	for i, jobs := range s.jobsAtTime {
		for k, job := range jobs {
			if shouldRemove(job) {
				jobs = removeAtIndex(jobs, k)
				s.jobsAtTime[i] = jobs
			}
		}
	}
}

func removeAtIndex(jobs JobSlice, i int) JobSlice {
	job := jobs[i]

	if i == len(jobs)-1 {
		return jobs[:i]
	}
	jobs = append(jobs[:i], jobs[i+1:]...)
	return jobs
}

// Scheduled checks if specific Job j was already added
func (s *Scheduler) Scheduled(j interface{}) bool {
	for _, job := range s.jobs {
		if job.jobFunc == getFunctionName(j) {
			return true
		}
	}
	return false
}

// Clear delete all scheduled Jobs
func (s *Scheduler) Clear() {
	s.jobs = newJobsSlice()
	//s.tickJobsMap = newSchedulableJobs() //FIXME
}

// Do specifies the jobFunc that should be called every time the Job runs
func (s *Scheduler) Do(jobFun interface{}, params ...interface{}) (*Job, error) {
	j := s.getCurrentJob()
	if j.err != nil {
		return nil, j.err
	}

	typ := reflect.TypeOf(jobFun)
	if typ.Kind() != reflect.Func {
		return nil, ErrNotAFunction
	}

	fname := getFunctionName(jobFun)
	j.funcs[fname] = jobFun
	j.fparams[fname] = params
	j.jobFunc = fname

	periodDuration, err := j.periodDuration()
	if err != nil {
		return nil, err
	}

	s.addJob(periodDuration, j)
	return j, nil
}

func (s *Scheduler) addJob(duration time.Duration, j *Job) {
	if _, exists := s.jobsAtTime[duration]; !exists {
		s.jobsAtTime[duration] = make(JobSlice, 0, 1)
	}
	s.jobsAtTime[duration] = append(s.jobsAtTime[duration], j)
}

// At schedules the Job at a specific time of day in the form "HH:MM:SS" or "HH:MM"
func (s *Scheduler) At(t string) *Scheduler {
	j := s.getCurrentJob()
	hour, min, sec, err := formatTime(t)
	if err != nil {
		j.err = ErrTimeFormat
		return s
	}
	// save atTime start as duration from midnight
	j.atTime = time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec)*time.Second
	return s
}

// AtTime schedules the next run of the Job
func (s *Scheduler) AtTime(t time.Time) *Scheduler {
	tStr := t.Format("15:04:05")
	s.At(tStr)
	return s
}

// StartImmediately sets the Jobs next run as soon as the scheduler starts
func (s *Scheduler) StartImmediately() *Scheduler {
	job := s.getCurrentJob()
	job.startsImmediately = true
	return s
}

// setUnit sets the unit type
func (s *Scheduler) setUnit(unit timeUnit) {
	currentJob := s.getCurrentJob()
	currentJob.unit = unit
}

// Second sets the unit with seconds
func (s *Scheduler) Second() *Scheduler {
	return s.Seconds()
}

// Seconds sets the unit with seconds
func (s *Scheduler) Seconds() *Scheduler {
	s.setUnit(seconds)
	return s
}

// Minute sets the unit with minutes
func (s *Scheduler) Minute() *Scheduler {
	return s.Minutes()
}

// Minutes sets the unit with minutes
func (s *Scheduler) Minutes() *Scheduler {
	s.setUnit(minutes)
	return s
}

// Hour sets the unit with hours
func (s *Scheduler) Hour() *Scheduler {
	return s.Hours()
}

// Hours sets the unit with hours
func (s *Scheduler) Hours() *Scheduler {
	s.setUnit(hours)
	return s
}

// Day sets the unit with days
func (s *Scheduler) Day() *Scheduler {
	s.setUnit(days)
	return s
}

// Days set the unit with days
func (s *Scheduler) Days() *Scheduler {
	s.setUnit(days)
	return s
}

// Week sets the unit with weeks
func (s *Scheduler) Week() *Scheduler {
	s.setUnit(weeks)
	return s
}

// Weeks sets the unit with weeks
func (s *Scheduler) Weeks() *Scheduler {
	s.setUnit(weeks)
	return s
}

// Weekday sets the start with a specific weekday weekday
func (s *Scheduler) Weekday(startDay time.Weekday) *Scheduler {
	s.getCurrentJob().startDay = startDay
	s.setUnit(weeks)
	return s
}

// Monday sets the start day as Monday
func (s *Scheduler) Monday() *Scheduler {
	return s.Weekday(time.Monday)
}

// Tuesday sets the start day as Tuesday
func (s *Scheduler) Tuesday() *Scheduler {
	return s.Weekday(time.Tuesday)
}

// Wednesday sets the start day as Wednesday
func (s *Scheduler) Wednesday() *Scheduler {
	return s.Weekday(time.Wednesday)
}

// Thursday sets the start day as Thursday
func (s *Scheduler) Thursday() *Scheduler {
	return s.Weekday(time.Thursday)
}

// Friday sets the start day as Friday
func (s *Scheduler) Friday() *Scheduler {
	return s.Weekday(time.Friday)
}

// Saturday sets the start day as Saturday
func (s *Scheduler) Saturday() *Scheduler {
	return s.Weekday(time.Saturday)
}

// Sunday sets the start day as Sunday
func (s *Scheduler) Sunday() *Scheduler {
	return s.Weekday(time.Sunday)
}

func (s *Scheduler) getCurrentJob() *Job {
	return s.jobs[len(s.jobs)-1]
}

// Lock prevents Job to run from multiple instances of gocron
func (s *Scheduler) Lock() *Scheduler {
	s.getCurrentJob().lock = true
	return s
}
