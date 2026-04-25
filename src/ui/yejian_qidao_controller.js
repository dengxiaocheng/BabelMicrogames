import {
  createInitialState,
  runCycle,
  checkEndings,
} from "../state/yejian_qidao_engine.js";

export class NightPrayerController {
  constructor(renderer, content) {
    this.renderer = renderer;
    this.config = content.config;
    this.characters = content.characters;
    this.events = content.events;
    this.state = createInitialState(this.config, this.characters);
    this._active = false;
  }

  start() {
    if (this._active) return;
    this._active = true;
    this.renderer.showStats();
    this._nextTurn();
  }

  getStatus() {
    const phase = this.config.nights.phases.find(
      (ph) => ph.id === this.state.currentPhaseId,
    );
    return {
      night: this.state.currentNight,
      phaseName: phase?.name ?? "终局",
      spirit: this.state.player.spirit,
      suspicion: this.state.player.suspicion,
      faith: this.state.player.faith,
    };
  }

  _nextTurn() {
    if (this.state.gameOver) {
      this.renderer.renderEnding(this.state.ending);
      return;
    }

    this.renderer.renderStatus(this.getStatus());

    const event = this._pickEvent();
    if (!event) {
      const ending = checkEndings(this.state, this.config);
      if (ending) {
        this.state = { ...this.state, gameOver: true, ending };
        this.renderer.renderEnding(ending);
      }
      return;
    }

    this.renderer.renderEvent(event, (choice) => this._onChoice(event, choice));
  }

  _pickEvent() {
    const pool = this.events.events.filter((e) =>
      e.phase.includes(this.state.currentPhaseId),
    );
    if (pool.length === 0 || this.state.gameOver) return null;
    return pool[this.state.eventIndex % pool.length];
  }

  _onChoice(event, choice) {
    const previousPhaseId = this.state.currentPhaseId;
    const result = runCycle(this.state, choice.effects, this.config);
    this.state = result.state;

    if (!this.state.gameOver && this.state.currentPhaseId === previousPhaseId) {
      this.state = { ...this.state, eventIndex: this.state.eventIndex + 1 };
    }

    this.renderer.renderFeedback(choice.feedback);
    this.renderer.renderStatus(this.getStatus());

    if (result.ending) {
      this.renderer.showContinue(() => {
        this.renderer.renderEnding(result.ending);
      });
      return;
    }

    if (this.state.currentPhaseId !== previousPhaseId) {
      const s = this.getStatus();
      this.renderer.renderPhaseTransition(`--- 进入 ${s.phaseName} ---`);
    }

    this.renderer.showContinue(() => this._nextTurn());
  }
}
