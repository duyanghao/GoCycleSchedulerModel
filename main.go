package main

import (
	"Scheduler"
	"log"
)

func main() {
	log.Printf("Start Cycle Scheduled Task...")
	// Construct CreateCycleScheduledTask
	// Week Cycle Scheduler Type
	/*createCycleScheduledTask := Scheduler.CreateCycleScheduledTask{
		CycleType:   0,
		CycleDay:    1,
		CycleHour:   12,
		CycleMinute: 30,
	}*/
	// Day Cycle Scheduler Type
	/*createCycleScheduledTask := Scheduler.CreateCycleScheduledTask{
		CycleType:   1,
		CycleDay:    1,
		CycleHour:   12,
		CycleMinute: 30,
	}*/
	// Hour Cycle Scheduler Type
	/*createCycleScheduledTask := Scheduler.CreateCycleScheduledTask{
		CycleType:   2,
		CycleDay:    1,
		CycleHour:   12,
		CycleMinute: 30,
	}*/
	// Minute Cycle Scheduler Type
	createCycleScheduledTask := Scheduler.CreateCycleScheduledTask{
		CycleType:   3,
		CycleDay:    1,
		CycleHour:   12,
		CycleMinute: 30,
	}
	schedulerTask := &Scheduler.SchedulerTaskChan{
		SchedulerTask: createCycleScheduledTask,
	}
	// Construct SchedulerWork
	schedulerWork := &Scheduler.SchedulerWork{}
	// Execute Work

	err := schedulerWork.Work(schedulerTask)
	if err != nil {
		log.Printf("schedulerWork Work error: %s", err)
	} else {
		for {
			//...
		}
	}
	log.Printf("End Cycle Scheduled Task")
	return
}
