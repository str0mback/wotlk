package druid

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
)

func (druid *Druid) registerBerserkCD() {
	if !druid.Talents.Berserk {
		return
	}

	actionId := core.ActionID{SpellID: 50334}
	glyphBonus := core.TernaryDuration(druid.HasMajorGlyph(proto.DruidMajorGlyph_GlyphOfBerserk), time.Second*5.0, 0.0)
	var affectedSpells []*core.Spell

	druid.BerserkAura = druid.RegisterAura(core.Aura{
		Label:    "Berserk",
		ActionID: actionId,
		Duration: (time.Second * 15) + glyphBonus,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = core.FilterSlice([]*core.Spell{
				druid.MangleCat,
				druid.FerociousBite,
				druid.Rake,
				druid.Rip,
				druid.SavageRoar,
				druid.SwipeCat,
				druid.Shred,
			}, func(spell *core.Spell) bool { return spell != nil })
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.CostMultiplier -= 0.5
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.CostMultiplier += 0.5
			}
		},
	})

	druid.Berserk = druid.RegisterSpell(core.SpellConfig{
		ActionID: actionId,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Minute * 3,
			},
			IgnoreHaste: true,
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.BerserkAura.Activate(sim)
		},
	})
}
