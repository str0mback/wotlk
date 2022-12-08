package dps

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/deathknight"
)

func (dk *DpsDeathknight) RotationActionCallback_BS_Frost(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false

	frAt := dk.NormalFrostRuneReadyAt(sim)
	uhAt := dk.NormalUnholyRuneReadyAt(sim)
	obAt := core.MaxDuration(frAt, uhAt)

	if !dk.fr.uaCycle {
		if sim.CurrentTime+1500*time.Millisecond <= obAt {
			casted = dk.FrostStrike.Cast(sim, target)
		}

		if !casted {
			dk.BloodStrike.Cast(sim, target)
			s.Advance()
		}
	} else {
		dk.BloodStrike.Cast(sim, target)
		s.Advance()
		dk.fr.uaCycle = false
	}

	return -1
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_Obli(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	advance := true
	waitTime := time.Duration(-1)

	ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
	bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()
	if sim.CurrentTime+1500*time.Millisecond < core.MinDuration(ffExpiresAt, bpExpiresAt) {
		if dk.Obliterate.CanCast(sim) {
			if dk.Deathchill != nil && dk.Deathchill.IsReady(sim) {
				dk.Deathchill.Cast(sim, target)
			}
			casted = dk.Obliterate.Cast(sim, target)
			advance = dk.LastOutcome.Matches(core.OutcomeLanded)
		}

		s.ConditionalAdvance(casted && advance)
	} else {
		waitTime = sim.CurrentTime
		s.Advance()
	}

	return core.TernaryDuration(casted, -1, waitTime)
}

func (dk *DpsDeathknight) RotationActionCallback_LastSecondsCast(sim *core.Simulation, target *core.Unit) bool {
	casted := false

	ffActive := dk.FrostFeverDisease[target.Index].IsActive()
	bpActive := dk.BloodPlagueDisease[target.Index].IsActive()
	ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
	bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()

	km := dk.KM()
	if core.MinDuration(ffExpiresAt, bpExpiresAt) > sim.CurrentTime+sim.GetRemainingDuration() {
		if dk.Obliterate.CanCast(sim) && ffActive && bpActive {
			if dk.Deathchill != nil && dk.Deathchill.IsReady(sim) {
				dk.Deathchill.Cast(sim, target)
			}
			casted = dk.Obliterate.Cast(sim, target)
		} else if dk.FrostStrike.CanCast(sim) && km {
			casted = dk.FrostStrike.Cast(sim, target)
		} else if dk.FrostStrike.CanCast(sim) {
			casted = dk.FrostStrike.Cast(sim, target)
		} else if dk.Obliterate.CanCast(sim) {
			if dk.Deathchill != nil && dk.Deathchill.IsReady(sim) {
				dk.Deathchill.Cast(sim, target)
			}
			casted = dk.Obliterate.Cast(sim, target)
		} else if dk.HowlingBlast.CanCast(sim) {
			casted = dk.HowlingBlast.Cast(sim, target)
		} else if dk.HornOfWinter.CanCast(sim) {
			casted = dk.HornOfWinter.Cast(sim, target)
		}
	}

	return casted
}

// TODO: Improve this
func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_FS_KM(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
	bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()

	casted = dk.RotationActionCallback_LastSecondsCast(sim, target)
	if !casted {
		km := dk.KM()
		if km && sim.CurrentTime+1500*time.Millisecond < core.MinDuration(ffExpiresAt, bpExpiresAt) {
			if dk.FrostStrike.CanCast(sim) {
				dk.FrostStrike.Cast(sim, target)
			}
		}

		s.Advance()
	}

	return core.TernaryDuration(casted, -1, sim.CurrentTime)
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_FS_Dump_UntilBR(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	waitUntil := time.Duration(-1)

	casted = dk.RotationActionCallback_LastSecondsCast(sim, target)
	if !casted {
		br := dk.CurrentBloodRunes() + dk.CurrentDeathRunes()
		ddAt := dk.SpentBloodRuneReadyAt()

		if br == 0 {
			spell := dk.RegularPrioPickSpell(sim, target, core.NeverExpires)
			if spell != nil {
				casted = spell.Cast(sim, target)
			} else {
				s.Advance()
				waitUntil = ddAt
			}
		} else if br == 1 {
			spell := dk.RegularPrioPickSpell(sim, target, ddAt+1500*time.Millisecond)
			if spell != nil {
				casted = spell.Cast(sim, target)
			} else {
				s.Advance()
				waitUntil = sim.CurrentTime
			}
		} else if br == 2 {
			ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
			bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()
			km := dk.KM()
			if km && sim.CurrentTime+1500*time.Millisecond < core.MinDuration(ffExpiresAt, bpExpiresAt) &&
				dk.DeathRuneRevertAt() > sim.CurrentTime+1500*time.Millisecond {
				if dk.FrostStrike.CanCast(sim) {
					casted = dk.FrostStrike.Cast(sim, target)
				}
			}

			s.Advance()
			waitUntil = sim.CurrentTime
		}
	}

	return core.TernaryDuration(casted, -1, waitUntil)
}
func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_FS_Dump_UntilUA(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	waitUntil := time.Duration(-1)

	casted = dk.RotationActionCallback_LastSecondsCast(sim, target)
	if !casted {
		if !dk.UnbreakableArmor.IsReady(sim) {
			spell := dk.RegularPrioPickSpell(sim, target, core.NeverExpires)
			if spell != nil {
				casted = spell.Cast(sim, target)
			} else {
				s.Advance()
				waitUntil = dk.UnbreakableArmor.ReadyAt()
			}
		} else {
			s.Advance()
			waitUntil = sim.CurrentTime
		}
	}

	return core.TernaryDuration(casted, -1, waitUntil)
}

func (dk *DpsDeathknight) getOblitDrift(sim *core.Simulation, castIn time.Duration) time.Duration {
	spendAt := sim.CurrentTime + castIn
	oblit1 := core.MinDuration(dk.RuneReadyAt(2), dk.RuneReadyAt(3))
	oblit2 := core.MinDuration(dk.SpendRuneReadyAt(4, spendAt), dk.SpendRuneReadyAt(5, spendAt))
	return oblit2 - oblit1
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_FS_Dump(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	waitUntil := time.Duration(-1)

	casted = dk.RotationActionCallback_LastSecondsCast(sim, target)
	if !casted {
		fr := dk.NormalCurrentFrostRunes()
		uh := dk.NormalCurrentUnholyRunes()
		frAt := dk.NormalFrostRuneReadyAt(sim)
		uhAt := dk.NormalUnholyRuneReadyAt(sim)
		obAt := core.MaxDuration(frAt, uhAt)
		abGCD := core.GCDDefault
		allowedObDrift := 3000 * time.Millisecond

		if fr == 2 && uh == 2 {
			casted := false
			waitTime := time.Duration(-1)

			ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
			bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()
			if sim.CurrentTime+1500*time.Millisecond < core.MinDuration(ffExpiresAt, bpExpiresAt) {
				if dk.Obliterate.CanCast(sim) {
					if dk.Deathchill != nil && dk.Deathchill.IsReady(sim) {
						dk.Deathchill.Cast(sim, target)
					}
					casted = dk.Obliterate.Cast(sim, target)
					if casted && dk.LastOutcome.Matches(core.OutcomeLanded) {
						dk.fr.oblitCount += 1
					}
				}
			} else {
				waitTime = sim.CurrentTime
				s.Advance()
			}

			return core.TernaryDuration(casted, -1, waitTime)
		} else if fr > 0 && uh > 0 {
			casted := false
			ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
			bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()
			km := dk.KM()
			if dk.fr.oblitCount == 1 && dk.FrostStrike.CanCast(sim) && km && sim.CurrentTime+1500*time.Millisecond < core.MinDuration(ffExpiresAt, bpExpiresAt) && dk.getOblitDrift(sim, abGCD) <= allowedObDrift {
				casted = dk.FrostStrike.Cast(sim, target)
			} else {
				if dk.Obliterate.CanCast(sim) {
					if dk.Deathchill != nil && dk.Deathchill.IsReady(sim) {
						dk.Deathchill.Cast(sim, target)
					}
					casted = dk.Obliterate.Cast(sim, target)
					advance := dk.LastOutcome.Matches(core.OutcomeLanded)
					if casted && advance {
						dk.fr.oblitCount += 1
					}

					if dk.fr.oblitCount == 2 {
						s.Advance()
						dk.fr.oblitCount = 0
						dk.fr.oblitDelay = sim.CurrentTime - dk.fr.oblitDelay
					}
				} else {
					s.Advance()
				}
			}

			return core.TernaryDuration(casted, -1, sim.CurrentTime)
		} else {
			spell := dk.RegularPrioPickSpell(sim, target, obAt+2500*time.Millisecond)
			if spell != nil {
				casted = spell.Cast(sim, target)
			} else {
				waitUntil = obAt
			}
		}
	}
	return core.TernaryDuration(casted, -1, waitUntil)
}

type FrostSubBloodUAState uint8

const (
	FrostSubBloodUAState_Now FrostSubBloodUAState = iota
	FrostSubBloodUAState_Soon
	FrostSubBloodUAState_CD
)

func (dk *DpsDeathknight) frCheckForUATime(sim *core.Simulation) FrostSubBloodUAState {
	if dk.UnbreakableArmor.IsReady(sim) {
		return FrostSubBloodUAState_Now
	} else {
		if dk.UnbreakableArmor.ReadyAt() < sim.CurrentTime+dk.SpellGCD()*2 {
			return FrostSubBloodUAState_Soon
		} else {
			return FrostSubBloodUAState_CD
		}
	}
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_UA_Check(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	uaState := dk.frCheckForUATime(sim)

	if uaState == FrostSubBloodUAState_CD {
		s.Clear().NewAction(dk.RotationActionCallback_FrostSubBlood_Obli_Check)
	} else {
		s.Clear().
			NewAction(dk.RotationActionCallback_FrostSubBlood_FS_Dump_UntilUA).
			NewAction(dk.RotationActionCallback_FrostSubBlood_FS_Dump_UntilBR).
			NewAction(dk.RotationActionCallback_UA_Frost).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
			NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
	}

	return sim.CurrentTime
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_Obli_Check(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	if dk.BloodTap.IsReady(sim) && dk.RuneIsDeath(0) && dk.RuneIsDeath(1) {
		s.Clear().
			NewAction(dk.RotationActionCallback_FrostSubBlood_FS_Dump_UntilBR).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
			NewAction(dk.RotationActionCallback_BT).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
			NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
	} else {
		s.Clear().
			NewAction(dk.RotationActionCallback_FrostSubBlood_FS_Dump_UntilBR).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
			NewAction(dk.RotationActionCallback_FrostSubBlood_FS_KM).
			NewAction(dk.RotationActionCallback_BS_Frost).
			NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
	}
	return sim.CurrentTime
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_SequenceRotation(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	s.Clear().
		NewAction(dk.RotationActionCallback_FrostSubBlood_FS_Dump)

	if dk.UnbreakableArmor != nil {
		s.NewAction(dk.RotationActionCallback_FrostSubBlood_UA_Check)
	} else {
		s.NewAction(dk.RotationActionCallback_FrostSubBlood_Obli_Check)
	}

	return sim.CurrentTime
}

func (dk *DpsDeathknight) setupFrostSubBloodERWOpener() {
	dk.setupUnbreakableArmorCooldowns()

	dk.RotationSequence.
		NewAction(dk.RotationActionCallback_IT).
		NewAction(dk.RotationActionCallback_PS).
		NewAction(dk.RotationActionCallback_UA_Frost).
		NewAction(dk.RotationActionCallback_BT).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_FS).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
		NewAction(dk.RotationActionCallback_ERW).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_RD).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_BS).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
}

func (dk *DpsDeathknight) setupFrostSubBloodNoERWOpener() {
	dk.setupUnbreakableArmorCooldowns()

	dk.RotationSequence.
		NewAction(dk.RotationActionCallback_IT).
		NewAction(dk.RotationActionCallback_PS).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_BS).
		NewAction(dk.RotationActionCallback_BT).
		NewAction(dk.RotationActionCallback_UA_Frost).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_RD).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_BS).
		NewAction(dk.RotationActionCallback_Frost_FS_HB).
		NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
		NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_RecoverFromPestiMiss(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	if dk.LastCast == dk.BloodStrike {
		s.Clear().
			NewAction(dk.RotationActionCallback_BS).
			NewAction(dk.RotationActionCallback_FS).
			NewAction(dk.RotationActionCallback_IT).
			NewAction(dk.RotationActionCallback_PS).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
			NewAction(dk.RotationActionCallback_BS).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
			NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
	} else {
		s.Clear().
			NewAction(dk.RotationActionCallback_BS).
			NewAction(dk.RotationActionCallback_BS).
			NewAction(dk.RotationActionCallback_FS).
			NewAction(dk.RotationActionCallback_IT).
			NewAction(dk.RotationActionCallback_PS).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Obli).
			NewAction(dk.RotationActionCallback_BS).
			NewAction(dk.RotationActionCallback_FrostSubBlood_Sequence_Pesti).
			NewAction(dk.RotationActionCallback_FrostSubBlood_SequenceRotation)
	}

	dk.NextCast = nil
	return sim.CurrentTime
}

func (dk *DpsDeathknight) RotationActionCallback_FrostSubBlood_Sequence_Pesti(sim *core.Simulation, target *core.Unit, s *deathknight.Sequence) time.Duration {
	casted := false
	waitUntil := time.Duration(-1)

	ffActive := dk.FrostFeverDisease[target.Index].IsActive()
	bpActive := dk.BloodPlagueDisease[target.Index].IsActive()
	ffExpiresAt := dk.FrostFeverDisease[target.Index].ExpiresAt()
	bpExpiresAt := dk.BloodPlagueDisease[target.Index].ExpiresAt()

	if dk.RotationActionCallback_LastSecondsCast(sim, target) {
		return -1
	}

	if !ffActive || !bpActive {
		return dk.RotationActionCallback_FrostSubBlood_RecoverFromPestiMiss(sim, target, s)
	} else {
		casted = dk.Pestilence.Cast(sim, target)
		advance := dk.LastOutcome.Matches(core.OutcomeLanded)
		if !casted || (casted && !dk.LastOutcome.Matches(core.OutcomeLanded)) {

			if sim.CurrentTime+dk.SpellGCD() > ffExpiresAt || sim.CurrentTime+dk.SpellGCD() > bpExpiresAt {
				return dk.RotationActionCallback_FrostSubBlood_RecoverFromPestiMiss(sim, target, s)
			} else {
				s.ConditionalAdvance(casted && advance)
				return core.TernaryDuration(casted, -1, waitUntil)
			}
		} else {
			s.ConditionalAdvance(casted && advance)
			return core.TernaryDuration(casted, -1, waitUntil)
		}
	}
}
