package gosfzplayer

// Voice represents an active playing voice/note
type Voice struct {
	sample     *Sample
	region     *SfzSection
	midiNote   uint8
	velocity   uint8
	position   float64 // Current playback position in samples (float for pitch adjustment)
	volume     float64
	pan        float64
	pitchRatio float64 // Pitch adjustment ratio (1.0 = no change, 2.0 = octave up)
	isActive   bool
	noteOn     bool
}
