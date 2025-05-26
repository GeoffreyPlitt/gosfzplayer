package gosfzplayer

import (
	"github.com/GeoffreyPlitt/debuggo"
)

var reverbDebug = debuggo.Debug("sfzplayer:reverb")

// Freeverb algorithm implementation
// Based on the classic Freeverb by Jezar at Dreampoint
// Optimized for real-time audio processing

const (
	// Freeverb constants
	numCombs     = 8
	numAllpasses = 4
	
	// Comb filter delays (in samples at 44.1kHz)
	combDelays = 8 * 1116 // Scale factor for different sample rates
	
	// Allpass filter delays
	allpassDelays = 8 * 556
	
	// Fixed point scaling
	fixedGain      = 0.015
	scaleWet       = 3.0
	scaleDry       = 2.0
	scaleDamp      = 0.4
	scaleRoom      = 0.28
	offsetRoom     = 0.7
	initialRoom    = 0.5
	initialDamp    = 0.5
	initialWet     = 1.0 / scaleWet
	initialDry     = 0.0
	initialWidth   = 1.0
	stereospread   = 23
)

// CombFilter implements a comb filter with damping
type CombFilter struct {
	buffer     []float64
	bufferSize int
	bufferIdx  int
	feedback   float64
	damp1      float64
	damp2      float64
	filterStore float64
}

// NewCombFilter creates a new comb filter
func NewCombFilter(size int) *CombFilter {
	return &CombFilter{
		buffer:     make([]float64, size),
		bufferSize: size,
		bufferIdx:  0,
		feedback:   0.0,
		damp1:      0.0,
		damp2:      0.0,
		filterStore: 0.0,
	}
}

// Process processes a sample through the comb filter
func (cf *CombFilter) Process(input float64) float64 {
	output := cf.buffer[cf.bufferIdx]
	
	// Apply damping filter
	cf.filterStore = (output * cf.damp2) + (cf.filterStore * cf.damp1)
	
	// Store new value with feedback
	cf.buffer[cf.bufferIdx] = input + (cf.filterStore * cf.feedback)
	
	// Advance buffer index
	cf.bufferIdx++
	if cf.bufferIdx >= cf.bufferSize {
		cf.bufferIdx = 0
	}
	
	return output
}

// SetDamp sets the damping parameters
func (cf *CombFilter) SetDamp(val float64) {
	cf.damp1 = val
	cf.damp2 = 1.0 - val
}

// SetFeedback sets the feedback amount
func (cf *CombFilter) SetFeedback(val float64) {
	cf.feedback = val
}

// AllpassFilter implements an allpass filter
type AllpassFilter struct {
	buffer     []float64
	bufferSize int
	bufferIdx  int
	feedback   float64
}

// NewAllpassFilter creates a new allpass filter
func NewAllpassFilter(size int) *AllpassFilter {
	return &AllpassFilter{
		buffer:     make([]float64, size),
		bufferSize: size,
		bufferIdx:  0,
		feedback:   0.5,
	}
}

// Process processes a sample through the allpass filter
func (af *AllpassFilter) Process(input float64) float64 {
	bufout := af.buffer[af.bufferIdx]
	output := -input + bufout
	af.buffer[af.bufferIdx] = input + (bufout * af.feedback)
	
	af.bufferIdx++
	if af.bufferIdx >= af.bufferSize {
		af.bufferIdx = 0
	}
	
	return output
}

// SetFeedback sets the feedback amount
func (af *AllpassFilter) SetFeedback(val float64) {
	af.feedback = val
}

// Freeverb implements the complete Freeverb algorithm
type Freeverb struct {
	// Filter banks for left and right channels
	combsL     [numCombs]*CombFilter
	combsR     [numCombs]*CombFilter
	allpassesL [numAllpasses]*AllpassFilter
	allpassesR [numAllpasses]*AllpassFilter
	
	// Parameters
	gain     float64
	roomSize float64
	damp     float64
	wet      float64
	dry      float64
	width    float64
	
	// Sample rate
	sampleRate int
}

// NewFreeverb creates a new Freeverb processor
func NewFreeverb(sampleRate int) *Freeverb {
	fv := &Freeverb{
		gain:       fixedGain,
		roomSize:   initialRoom,
		damp:       initialDamp,
		wet:        initialWet,
		dry:        initialDry,
		width:      initialWidth,
		sampleRate: sampleRate,
	}
	
	// Calculate delay lengths based on sample rate
	scaleFactor := float64(sampleRate) / 44100.0
	
	// Initialize comb filters
	combDelayLengths := []int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617}
	for i := 0; i < numCombs; i++ {
		delayL := int(float64(combDelayLengths[i]) * scaleFactor)
		delayR := delayL + stereospread
		fv.combsL[i] = NewCombFilter(delayL)
		fv.combsR[i] = NewCombFilter(delayR)
	}
	
	// Initialize allpass filters
	allpassDelayLengths := []int{556, 441, 341, 225}
	for i := 0; i < numAllpasses; i++ {
		delayL := int(float64(allpassDelayLengths[i]) * scaleFactor)
		delayR := delayL + stereospread
		fv.allpassesL[i] = NewAllpassFilter(delayL)
		fv.allpassesR[i] = NewAllpassFilter(delayR)
	}
	
	// Set initial parameters
	fv.updateParameters()
	
	reverbDebug("Freeverb initialized: sampleRate=%d, scaleFactor=%.2f", sampleRate, scaleFactor)
	return fv
}

// updateParameters updates all filter parameters
func (fv *Freeverb) updateParameters() {
	// Calculate room size parameter
	roomScaled := (fv.roomSize * scaleRoom) + offsetRoom
	
	// Calculate damping
	dampScaled := fv.damp * scaleDamp
	
	// Update all comb filters
	for i := 0; i < numCombs; i++ {
		fv.combsL[i].SetFeedback(roomScaled)
		fv.combsR[i].SetFeedback(roomScaled)
		fv.combsL[i].SetDamp(dampScaled)
		fv.combsR[i].SetDamp(dampScaled)
	}
	
	// Allpass filters have fixed feedback
	for i := 0; i < numAllpasses; i++ {
		fv.allpassesL[i].SetFeedback(0.5)
		fv.allpassesR[i].SetFeedback(0.5)
	}
}

// SetRoomSize sets the room size (0.0 to 1.0)
func (fv *Freeverb) SetRoomSize(size float64) {
	if size < 0.0 {
		size = 0.0
	}
	if size > 1.0 {
		size = 1.0
	}
	fv.roomSize = size
	fv.updateParameters()
}

// SetDamping sets the damping amount (0.0 to 1.0)
func (fv *Freeverb) SetDamping(damp float64) {
	if damp < 0.0 {
		damp = 0.0
	}
	if damp > 1.0 {
		damp = 1.0
	}
	fv.damp = damp
	fv.updateParameters()
}

// SetWet sets the wet level (0.0 to 1.0)
func (fv *Freeverb) SetWet(wet float64) {
	if wet < 0.0 {
		wet = 0.0
	}
	if wet > 1.0 {
		wet = 1.0
	}
	fv.wet = wet * scaleWet
}

// SetDry sets the dry level (0.0 to 1.0)
func (fv *Freeverb) SetDry(dry float64) {
	if dry < 0.0 {
		dry = 0.0
	}
	if dry > 1.0 {
		dry = 1.0
	}
	fv.dry = dry * scaleDry
}

// SetWidth sets the stereo width (0.0 to 1.0)
func (fv *Freeverb) SetWidth(width float64) {
	if width < 0.0 {
		width = 0.0
	}
	if width > 1.0 {
		width = 1.0
	}
	fv.width = width
}

// ProcessStereo processes a stereo sample pair through the reverb
func (fv *Freeverb) ProcessStereo(inputL, inputR float64) (outputL, outputR float64) {
	// Scale input
	input := (inputL + inputR) * fv.gain
	
	// Process through comb filters
	var outL, outR float64
	for i := 0; i < numCombs; i++ {
		outL += fv.combsL[i].Process(input)
		outR += fv.combsR[i].Process(input)
	}
	
	// Process through allpass filters
	for i := 0; i < numAllpasses; i++ {
		outL = fv.allpassesL[i].Process(outL)
		outR = fv.allpassesR[i].Process(outR)
	}
	
	// Apply wet/dry mix and stereo width
	wetL := outL*fv.wet
	wetR := outR*fv.wet
	
	// Stereo width processing
	wet1 := wetL * (fv.width/2.0 + 0.5)
	wet2 := wetR * ((1.0-fv.width)/2.0)
	
	outputL = (inputL * fv.dry) + wet1 + wet2
	outputR = (inputR * fv.dry) + wet1 + wet2
	
	return outputL, outputR
}

// ProcessMono processes a mono sample through the reverb
func (fv *Freeverb) ProcessMono(input float64) float64 {
	outL, _ := fv.ProcessStereo(input, input)
	return outL
}

// GetRoomSize returns the current room size
func (fv *Freeverb) GetRoomSize() float64 {
	return fv.roomSize
}

// GetDamping returns the current damping
func (fv *Freeverb) GetDamping() float64 {
	return fv.damp
}

// GetWet returns the current wet level
func (fv *Freeverb) GetWet() float64 {
	return fv.wet / scaleWet
}

// GetDry returns the current dry level
func (fv *Freeverb) GetDry() float64 {
	return fv.dry / scaleDry
}

// GetWidth returns the current stereo width
func (fv *Freeverb) GetWidth() float64 {
	return fv.width
}