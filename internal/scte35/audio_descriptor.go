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

// AudioDescriptorTag is the splice_descriptor_tag for an audio descriptor.
const AudioDescriptorTag = 0x04

// AudioDescriptor is an implementation of a audio_descriptor. The
// audio_descriptor() should be used when programmers and/or MVPDs do not
// support dynamic signaling (e.g., signaling of audio language changes) and
// with legacy audio formats that do not support dynamic signaling.
type AudioDescriptor struct {
	AudioChannels []AudioChannel
}

// Tag returns the splice_descriptor_tag.
func (sd *AudioDescriptor) Tag() uint32 { return AudioDescriptorTag }

// decode updates this SpliceDescriptor from binary.
func (sd *AudioDescriptor) decode(b []byte) error {
	r := iobit.NewReader(b)
	r.Skip(8)  // splice_descriptor_tag
	r.Skip(8)  // descriptor_length
	r.Skip(32) // identifier
	audioCount := int(r.Uint32(4))
	r.Skip(4) // reserved
	sd.AudioChannels = make([]AudioChannel, audioCount)
	for i := 0; i < audioCount; i++ {
		ac := AudioChannel{}
		ac.ComponentTag = r.Uint32(8)
		ac.ISOCode = r.String(3)
		ac.BitStreamMode = r.Uint32(3)
		ac.NumChannels = r.Uint32(4)
		ac.FullSrvcAudio = r.Bit()
		sd.AudioChannels[i] = ac
	}

	if err := readerError(r); err != nil {
		return fmt.Errorf("audio_descriptor: %w", err)
	}
	return nil
}

// encode this SpliceDescriptor to binary.
func (sd *AudioDescriptor) encode() ([]byte, error) {
	length := sd.length()

	// add 2 bytes to contain splice_descriptor_tag & descriptor_length
	buf := make([]byte, length+2)
	iow := iobit.NewWriter(buf)
	iow.PutUint32(8, AudioDescriptorTag)
	iow.PutUint32(8, uint32(length))
	iow.PutUint32(32, CUEIdentifier)
	iow.PutUint32(8, uint32(len(sd.AudioChannels)))
	iow.PutUint32(4, Reserved)
	for _, ad := range sd.AudioChannels {
		iow.PutUint32(8, ad.ComponentTag)
		_, _ = iow.Write([]byte(ad.ISOCode))
		iow.PutUint32(3, ad.BitStreamMode)
		iow.PutUint32(4, ad.NumChannels)
		iow.PutBit(ad.FullSrvcAudio)
	}
	return buf, nil
}

// descriptorLength returns the descriptor_length
func (sd *AudioDescriptor) length() int {
	length := 32 // identifier
	length += 4  // audio_count
	length += 4  // reserved
	for i := range sd.AudioChannels {
		length += sd.AudioChannels[i].length() * 8
	}
	return length / 8
}

// AudioChannel collects the audio PID details.
type AudioChannel struct {
	ComponentTag  uint32
	ISOCode       string
	BitStreamMode uint32
	NumChannels   uint32
	FullSrvcAudio bool
}

// length returns audio_channel length.
func (ac *AudioChannel) length() int {
	length := 8  // component_tag
	length += 24 // iso_code
	length += 3  // bit_stream_mode
	length += 4  // num_channels
	length++     // full_srvc_audio
	return length / 8
}
