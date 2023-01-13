package dps

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/deathknight"
)

func (dk *DpsDeathknight) setupBloodRotations() {
	dk.Inputs.FuStrike = deathknight.FuStrike_DeathStrike
	if dk.Talents.Annihilation > 0 {
		dk.Inputs.FuStrike = deathknight.FuStrike_Obliterate
	}

	dk.setupDrwCooldowns()

	dk.RotationSequence.Clear().
		NewAction(dk.RotationActionCallback_IT).
		NewAction(dk.RotationActionCallback_PS).
		NewAction(dk.RotationActionCallback_HS).
		NewAction(dk.RotationActionCallback_FU).
		NewAction(dk.RotationActionCallback_HS).
		NewAction(dk.RotationActionCallback_ERW).
		NewAction(dk.RotationActionCallback_RD).
		NewAction(dk.RotationActionCallback_DRW).
		NewAction(dk.RotationActionCallback_IT).
		NewAction(dk.RotationActionCallback_PS).
		NewAction(dk.RotationActionCallback_HS).
		NewAction(dk.RotationActionCallback_HS).
		NewAction(dk.RotationActionCallback_HS).
		NewAction(dk.RotationActionCallback_HS)

	dk.RotationSequence.NewAction(dk.RotationActionCallback_BloodRotation)
}

func (dk *DpsDeathknight) RotationActionCallback_BloodRotation(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false

	if dk.blDrwCheck(sim, target, 100*time.Millisecond) {
		if dk.DancingRuneWeapon.IsReady(sim) {
			dk.RotationSequence.Clear()

			if dk.UnholyFrenzy.IsReady(sim) {
				dk.RotationSequence.NewAction(dk.RotationActionCallback_UF)
			}

			dk.RotationSequence.
				NewAction(dk.RotationActionCallback_DRW).
				NewAction(dk.RotationActionBL_IT_Custom).
				NewAction(dk.RotationActionBL_PS_Custom).
				NewAction(dk.RotationAction_ResetToBloodMain)
		} else {
			dk.blAfterDrwSequence(sim)
		}
		return sim.CurrentTime
	}

	if dk.RaiseDead.CanCast(sim) && sim.GetRemainingDuration() >= time.Second*30 {
		dk.RaiseDead.Cast(sim, target)
		return sim.CurrentTime
	}

	fuStrike := dk.DeathStrike
	if dk.Inputs.FuStrike == deathknight.FuStrike_Obliterate {
		fuStrike = dk.Obliterate
	}

	if !casted {
		if dk.blDiseaseCheck(sim, target, dk.BloodStrike, true, 1) {
			if dk.shShouldSpreadDisease(sim) {
				return dk.blSpreadDiseases(sim, target, s)
			} else {
				if dk.Talents.HeartStrike {
					casted = dk.HeartStrike.Cast(sim, target)
				} else {
					casted = dk.BloodStrike.Cast(sim, target)
				}
			}
		} else {
			dk.blRecastDiseasesSequence(sim)
			return sim.CurrentTime
		}
		if !casted {
			if dk.blDiseaseCheck(sim, target, fuStrike, true, 1) {
				casted = fuStrike.Cast(sim, target)
			} else {
				dk.blRecastDiseasesSequence(sim)
				return sim.CurrentTime
			}
			if !casted {
				if dk.blDeathCoilCheck(sim) {
					casted = dk.DeathCoil.Cast(sim, target)
				}
				if !casted && dk.HornOfWinter.CanCast(sim) {
					dk.HornOfWinter.Cast(sim, target)
				}
			}
		}
	}

	return -1
}

func (dk *DpsDeathknight) blAfterDrwSequence(sim *core.Simulation) {
	dk.RotationSequence.Clear()

	dk.RotationSequence.
		NewAction(dk.RotationActionBL_IT_Custom).
		NewAction(dk.RotationActionBL_PS_Custom).
		NewAction(dk.RotationAction_ResetToBloodMain)
}

func (dk *DpsDeathknight) blRecastDiseasesSequence(sim *core.Simulation) {
	dk.RotationSequence.Clear()

	// If we have glyph of Disease and both dots active try to refresh with pesti
	didPesti := false
	if dk.sr.hasGod {
		if dk.FrostFeverDisease[dk.CurrentTarget.Index].IsActive() && dk.BloodPlagueDisease[dk.CurrentTarget.Index].IsActive() {
			didPesti = true
			dk.RotationSequence.NewAction(dk.RotationActionCallback_Pesti_Custom)
		}
	}

	// If we did not pesti queue normal dot refresh
	if !didPesti {
		dk.RotationSequence.
			NewAction(dk.RotationActionBL_FF_ClipCheck).
			NewAction(dk.RotationActionBL_IT_Custom).
			NewAction(dk.RotationActionBL_BP_ClipCheck).
			NewAction(dk.RotationActionBL_PS_Custom)
	}

	dk.RotationSequence.
		NewAction(dk.RotationAction_ResetToBloodMain)
}

func (dk *DpsDeathknight) RotationAction_ResetToBloodMain(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	dk.RotationSequence.Clear().
		NewAction(dk.RotationActionCallback_BloodRotation)

	return sim.CurrentTime
}

func (dk *DpsDeathknight) RotationActionCallback_FU(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	if dk.Inputs.FuStrike == deathknight.FuStrike_DeathStrike {
		casted = dk.DeathStrike.Cast(sim, target)
	} else if dk.Inputs.FuStrike == deathknight.FuStrike_Obliterate {
		casted = dk.Obliterate.Cast(sim, target)
	}
	advance := dk.LastOutcome.Matches(core.OutcomeLanded)

	s.ConditionalAdvance(casted && advance)
	return -1
}

// Custom PS callback for tracking recasts for pestilence disease sync
func (dk *DpsDeathknight) RotationActionBL_PS_Custom(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := dk.PlagueStrike.Cast(sim, target)
	advance := dk.LastOutcome.Matches(core.OutcomeLanded)

	dk.sr.recastedBP = casted && advance
	s.ConditionalAdvance(casted && advance)
	return -1
}

// Custom IT callback for tracking recasts for pestilence disease sync
func (dk *DpsDeathknight) RotationActionBL_IT_Custom(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := dk.IcyTouch.Cast(sim, target)
	advance := dk.LastOutcome.Matches(core.OutcomeLanded)
	dk.sr.recastedFF = casted && advance
	s.ConditionalAdvance(casted && advance)
	return -1
}

func (dk *DpsDeathknight) RotationActionBL_FF_ClipCheck(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	dot := dk.FrostFeverDisease[target.Index]
	gracePeriod := dk.CurrentFrostRuneGrace(sim)
	return dk.RotationActionBL_DiseaseClipCheck(dot, gracePeriod, sim, target, s)
}

func (dk *DpsDeathknight) RotationActionBL_BP_ClipCheck(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	dot := dk.BloodPlagueDisease[target.Index]
	gracePeriod := dk.CurrentUnholyRuneGrace(sim)
	return dk.RotationActionBL_DiseaseClipCheck(dot, gracePeriod, sim, target, s)
}

// Check if we have enough rune grace period to delay the disease cast
// so we get more ticks without losing on rune cd
func (dk *DpsDeathknight) RotationActionBL_DiseaseClipCheck(dot *core.Dot, gracePeriod time.Duration, sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	// TODO: Play around with allowing rune cd to be wasted
	// for more disease ticks and see if its a worth option for the ui
	//runeCdWaste := 0 * time.Millisecond
	var waitUntil time.Duration
	if dot.TickCount < dot.NumberOfTicks-1 {
		nextTickAt := dot.ExpiresAt() - dot.TickLength*time.Duration((dot.NumberOfTicks-1)-dot.TickCount)
		if nextTickAt > sim.CurrentTime && (nextTickAt < sim.CurrentTime+gracePeriod || nextTickAt < sim.CurrentTime+400*time.Millisecond) {
			// Delay disease for next tick
			dk.LastOutcome = core.OutcomeMiss
			waitUntil = nextTickAt + 50*time.Millisecond
		} else {
			waitUntil = sim.CurrentTime
		}
	} else {
		waitUntil = sim.CurrentTime
	}

	s.Advance()
	return waitUntil
}
