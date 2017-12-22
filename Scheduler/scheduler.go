package Scheduler

import (
	"Scheduler/utils/iso8601"
	"fmt"
	"log"
	"sync"
	"time"
)

type SignatureSchedule struct {
	// ISO 8601 String
	// e.g. "R/2014-03-08T20:00:00.000Z/PT2H"
	Schedule string
	// Runed count of the task
	RepeatedCount int64

	runTimer *time.Timer
}

func (s *SignatureSchedule) StopRunTimer() bool {
	if s.runTimer != nil {
		return s.runTimer.Stop()
	}
	return true
}

func (s *SignatureSchedule) ResetRunTimer(d time.Duration) bool {
	if s.runTimer != nil {
		return s.runTimer.Reset(d)
	}
	return false
}

func (s *SignatureSchedule) SetRunTimer(d time.Duration, f func()) bool {
	if s.runTimer != nil {
		return false
	}
	s.runTimer = time.AfterFunc(d, f)
	return true
}

type CreateCycleScheduledTask struct {
	CycleType   int // represents cycle Scheduler type(week, day, hour and minute currently supported)
	CycleDay    int // represents week
	CycleHour   int // represents hour
	CycleMinute int // represents minute
}

type SchedulerTaskChan struct {
	SchedulerTask CreateCycleScheduledTask
	ETA           *time.Time
	// for schedule task
	Schedule *SignatureSchedule
	ErrChan  chan error
}

type SchedulerWork struct {
	sync.RWMutex
}

func CompareTime(sTime time.Time, cycleDay, cycleHour, cycleMinute int) bool {
	today := sTime.AddDate(0, 0, 0).Format("2006-01-02")
	td, _ := time.Parse("2006-01-02", today)
	td = td.Add(time.Duration(cycleHour) * time.Hour)
	td = td.Add(time.Duration(cycleMinute) * time.Minute)
	now := sTime.Format("2006-01-02 15:04:05")
	td2, _ := time.Parse("2006-01-02 15:04:05", now)

	if td.After(td2) {
		return true
	} else {
		return false
	}
}

func CalStartTime(sTime time.Time, cycleDay, cycleHour, cycleMinute int) string {
	tomorrow := sTime.AddDate(0, 0, cycleDay).Format("2006-01-02")
	td, _ := time.Parse("2006-01-02", tomorrow)
	td = td.Add(time.Duration(cycleHour) * time.Hour)
	td = td.Add(time.Duration(cycleMinute) * time.Minute)
	st := td.Format(iso8601.RFC3339WithoutTimezone)
	st += "+08:00"
	return st
}

func ConvertNotation(sTime time.Time, cycleType, cycleDay, cycleHour, cycleMinute int) *SignatureSchedule {
	var rc, st, dur string
	// for duration and start_time
	switch cycleType {
	case 0: // represents week cycle type
		dur = "P1W"
		weekday := int(sTime.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if cycleDay == 0 {
			cycleDay = 7
		}
		if weekday == cycleDay {
			isexec := CompareTime(sTime, 0, cycleHour, cycleMinute)
			if isexec {
				st = CalStartTime(sTime, 0, cycleHour, cycleMinute)
			} else {
				st = CalStartTime(sTime, 7, cycleHour, cycleMinute)
			}
		} else if weekday < cycleDay {
			daydur := cycleDay - weekday
			st = CalStartTime(sTime, daydur, cycleHour, cycleMinute)
		} else {
			daydur := 7 - weekday + cycleDay
			st = CalStartTime(sTime, daydur, cycleHour, cycleMinute)
		}
	case 1: // represents day cycle type
		dur = "P1D"
		isexec := CompareTime(sTime, 0, cycleHour, cycleMinute)
		if isexec {
			st = CalStartTime(sTime, 0, cycleHour, cycleMinute)
		} else {
			st = CalStartTime(sTime, 1, cycleHour, cycleMinute)
		}
	case 2: // represents hour cycle type
		dur = "PT1H"
		nextTime := sTime.Add(time.Hour)
		st = CalStartTime(nextTime, 0, nextTime.Hour(), 0)
	case 3: // represents minute cycle type
		dur = "PT1M"
		nextTime := sTime.Add(time.Minute)
		st = CalStartTime(nextTime, 0, nextTime.Hour(), nextTime.Minute())
	}

	// for repeatedCount
	rc = "R"

	return &SignatureSchedule{
		Schedule:      rc + "/" + st + "/" + dur,
		RepeatedCount: 0,
		runTimer:      nil,
	}
}

// execute cycle scheduled task...
func (sw *SchedulerWork) emit(schedulertask *SchedulerTaskChan) error {
	schedulertask.Schedule.RepeatedCount++
	repeatedCount := schedulertask.Schedule.RepeatedCount
	log.Printf("Task is repeating the %d time...", repeatedCount)

	// set next call
	notation := schedulertask.Schedule.Schedule

	rc, _, dur, err := iso8601.Parse(notation)
	if err != nil {
		// stop
		schedulertask.Schedule.StopRunTimer()
		return fmt.Errorf("Error to parse schedule time: %s", notation)
	}
	if rc > 0 && repeatedCount >= rc {
		log.Printf("Task has finished to run %d times", repeatedCount)

		// stop
		schedulertask.Schedule.StopRunTimer()

		return nil
	}

	prevETA := *schedulertask.ETA
	nextETA := prevETA.UTC().Add(dur)
	now := time.Now().UTC()
	if nextETA.After(now) {
		schedulertask.ETA = &nextETA
		waitDuration := nextETA.Sub(now)
		schedulertask.Schedule.ResetRunTimer(waitDuration)
		log.Printf("Next Start Time: %s,and need to wait for: %s...", *schedulertask.ETA, waitDuration)
	} else {
		// stop
		schedulertask.Schedule.StopRunTimer()
		return fmt.Errorf("Task is not scheduled in %s", dur.String())
	}

	return nil
}

func (sw *SchedulerWork) Work(schedulertask *SchedulerTaskChan) error {
	if schedulertask.Schedule == nil {
		schedulertask.Schedule = ConvertNotation(time.Now(), schedulertask.SchedulerTask.CycleType, schedulertask.SchedulerTask.CycleDay, schedulertask.SchedulerTask.CycleHour, schedulertask.SchedulerTask.CycleMinute)
	}
	notation := schedulertask.Schedule.Schedule
	repeatedCount := schedulertask.Schedule.RepeatedCount
	rc, ts, dur, err := iso8601.Parse(notation)
	if err != nil {
		return fmt.Errorf("Failed to parse schedule time: %s, error: %s", notation, err)
	}
	timerFunc := func() {
		log.Printf("Emit schedule task: %+v ...", schedulertask.SchedulerTask)

		err := sw.emit(schedulertask)

		if err != nil {
			log.Printf("Failed to emit schedule task: %+v, error: %s", schedulertask.SchedulerTask, err)
			return
		}

		log.Printf("Emit schedule task: %+v Successfully", schedulertask.SchedulerTask)
	}

	if schedulertask.ETA == nil {
		utcTs := ts.UTC()
		schedulertask.ETA = &utcTs
	}

	now := time.Now().UTC()
	nextETA := *schedulertask.ETA
	if rc == -1 || repeatedCount < rc {
		if nextETA.After(now) {
			schedulertask.ETA = &nextETA
			waitDuration := nextETA.Sub(now)
			schedulertask.Schedule.SetRunTimer(waitDuration, timerFunc)
			log.Printf("Next Start Time: %s,and need to wait for: %s...", *schedulertask.ETA, waitDuration)
			log.Printf("Start Cycle schedule task at: %+v(repeatedCount: %d and Interval: %s) ...", ts, rc, dur.String())
			return nil
		} else {
			return fmt.Errorf("nextETA: %v is before now: %v", nextETA, now)
		}
	} else {
		return fmt.Errorf("invalid rc: %d or repeatedCount: %d", rc, repeatedCount)
	}
}
