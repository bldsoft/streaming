// Copyright 2021 Comcast Cable Communications Management, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or   implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package scte35

import (
	"fmt"

	"github.com/bamiaux/iobit"
)

// SpliceScheduleType is the splice_command_type for the splice_schedule()
// command.
const SpliceScheduleType = 0x04

// SpliceSchedule is provided to allow a schedule of splice events to be
// conveyed in advance.
type SpliceSchedule struct {
	Events []Event
}

// Type returns the splice_command_type
func (cmd *SpliceSchedule) Type() uint32 { return SpliceScheduleType }

// decode a binary splice_schedule.
func (cmd *SpliceSchedule) decode(b []byte) error {
	r := iobit.NewReader(b)

	spliceCount := int(r.Uint32(8))
	cmd.Events = make([]Event, spliceCount)
	for i := 0; i < spliceCount; i++ {
		e := Event{}
		e.SpliceEventID = r.Uint32(32)
		e.SpliceEventCancelIndicator = r.Bit()
		if !e.SpliceEventCancelIndicator {
			e.OutOfNetworkIndicator = r.Bit()
			programSpliceFlag := r.Bit()
			durationFlag := r.Bit()
			r.Skip(5) // reserved
			if programSpliceFlag {
				e.Program = &EventProgram{}
				e.Program.UTCSpliceTime = NewUTCSpliceTime(r.Uint32(32))
			} else {
				componentCount := int(r.Uint32(8))
				e.Components = make([]EventComponent, componentCount)
				for j := 0; j < componentCount; j++ {
					c := EventComponent{}
					c.Tag = r.Uint32(8)
					c.UTCSpliceTime = NewUTCSpliceTime(r.Uint32(32))
					e.Components[j] = c
				}
			}
			if durationFlag {
				e.BreakDuration = &BreakDuration{}
				e.BreakDuration.AutoReturn = r.Bit()
				r.Skip(6) // reserved
				e.BreakDuration.Duration = r.Uint64(33)
			}
		}
		e.UniqueProgramID = r.Uint32(16)
		e.AvailNum = r.Uint32(8)
		e.AvailsExpected = r.Uint32(8)
		cmd.Events[i] = e
	}

	if err := readerError(r); err != nil {
		return fmt.Errorf("splice_schedule: %w", err)
	}
	return nil
}

// encode this splice_schedule to binary.
func (cmd *SpliceSchedule) encode() ([]byte, error) {
	buf := make([]byte, cmd.length())
	iow := iobit.NewWriter(buf)

	iow.PutUint32(8, uint32(len(cmd.Events)))
	for _, e := range cmd.Events {
		iow.PutUint32(32, e.SpliceEventID)
		iow.PutBit(e.SpliceEventCancelIndicator)
		iow.PutUint32(7, Reserved) // reserved
		if !e.SpliceEventCancelIndicator {
			iow.PutBit(e.OutOfNetworkIndicator)
			iow.PutBit(e.ProgramSpliceFlag())
			iow.PutBit(e.DurationFlag())
			iow.PutUint32(5, Reserved) // reserved
			if e.ProgramSpliceFlag() {
				iow.PutUint32(32, e.Program.UTCSpliceTime.GPSSeconds())
			} else {
				iow.PutUint32(8, uint32(len(e.Components)))
				for _, c := range e.Components {
					iow.PutUint32(8, c.Tag)
					iow.PutUint32(32, c.UTCSpliceTime.GPSSeconds())
				}
			}
			if e.DurationFlag() {
				iow.PutBit(e.BreakDuration.AutoReturn)
				iow.PutUint32(6, Reserved)
				iow.PutUint64(33, e.BreakDuration.Duration)
			}
		}
		iow.PutUint32(16, e.UniqueProgramID)
		iow.PutUint32(8, e.AvailNum)
		iow.PutUint32(8, e.AvailsExpected)
	}

	return buf, iow.Flush()
}

// commandLength returns the splice_command_length
func (cmd SpliceSchedule) length() int {
	length := 8 // splice_count

	// for i = 0 to splice_count
	for _, e := range cmd.Events {
		length += 32 // splice_event_id
		length++     // splice_event_cancel_indicator
		length += 7  // reserved

		// if splice_event_cancel_indicator == 0
		if !e.SpliceEventCancelIndicator {
			length++    // out_of_network_indicator
			length++    // program_splice_flag
			length++    // duration_flag
			length += 5 // reserved

			if e.ProgramSpliceFlag() {
				// program_splice_flag == 1
				length += 32 // utc_splice_time
			} else {
				// program_splice_flag == 0
				length += 8 // component_count
				for range e.Components {
					length += 8  // component_tag
					length += 32 // utc_splice_time
				}
			}

			// if duration_flag == 1
			if e.DurationFlag() {
				length++     // auto_return
				length += 6  // reserved
				length += 33 // duration
			}

			length += 16 // unique_program_id
			length += 8  // avail_num
			length += 8  // avails_expected
		}
	}

	return length / 8
}

// Event is a single event within a splice_schedule.
type Event struct {
	Program                    *EventProgram
	Components                 []EventComponent
	BreakDuration              *BreakDuration
	SpliceEventID              uint32
	SpliceEventCancelIndicator bool
	OutOfNetworkIndicator      bool
	UniqueProgramID            uint32
	AvailNum                   uint32
	AvailsExpected             uint32
}

// DurationFlag returns the duration_flag.
func (e *Event) DurationFlag() bool {
	return e != nil && e.BreakDuration != nil
}

// ProgramSpliceFlag returns the program_splice_flag.
func (e *Event) ProgramSpliceFlag() bool {
	return e != nil && e.Program != nil
}

// EventComponent contains the Splice Points in Component Splice Mode.
type EventComponent struct {
	Tag           uint32
	UTCSpliceTime UTCSpliceTime
}

// EventProgram contains the Splice Point in Program Splice Mode
type EventProgram struct {
	UTCSpliceTime UTCSpliceTime
}
