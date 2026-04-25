export class NightPrayerRenderer {
  constructor(el) {
    this.el = el;
    this._onChoose = null;
  }

  renderStatus(status) {
    this.el.night.textContent    = `夜 ${status.night}`;
    this.el.phase.textContent    = status.phaseName;
    this.el.spirit.textContent   = `精神 ${status.spirit}`;
    this.el.suspicion.textContent = `怀疑 ${status.suspicion}`;
    this.el.faith.textContent    = `信仰 ${status.faith}`;
  }

  showStats()  { this._setVisible(this.el, ["night","spirit","suspicion","faith"], true); }
  hideStats()  { this._setVisible(this.el, ["night","spirit","suspicion","faith"], false); }

  renderEvent(event, onChoose) {
    this._show(this.el.eventTitle);
    this.el.eventTitle.textContent = event.title;
    this.el.narrative.textContent  = event.narrative;
    this._show(this.el.narrative);
    this._hide(this.el.feedback);
    this._hide(this.el.ending);
    this._hide(this.el.phaseTransition);
    this._renderChoices(event.choices, onChoose);
  }

  _renderChoices(choices, onChoose) {
    this.el.choices.innerHTML = "";
    this._onChoose = onChoose;

    choices.forEach((choice, i) => {
      const btn = document.createElement("button");
      btn.className = "choice-btn";
      btn.textContent = `[${i + 1}] ${choice.text}`;
      btn.addEventListener("click", () => this._select(choices, i));
      this.el.choices.appendChild(btn);
    });
  }

  _select(choices, idx) {
    const btns = this.el.choices.querySelectorAll(".choice-btn");
    btns.forEach((btn, i) => {
      if (i === idx) btn.classList.add("selected");
      btn.disabled = true;
    });
    if (this._onChoose) this._onChoose(choices[idx]);
  }

  renderFeedback(text) {
    this.el.feedback.textContent = text;
    this.el.feedback.style.display = "";
    this.el.feedback.className = "visible";
  }

  renderPhaseTransition(text) {
    this.el.phaseTransition.textContent = text;
    this.el.phaseTransition.style.display = "";
    this.el.phaseTransition.className = "visible";
  }

  showContinue(onContinue) {
    const prompt = document.createElement("div");
    prompt.className = "continue-prompt";
    prompt.textContent = "-- 点击继续 --";
    prompt.addEventListener("click", () => { prompt.remove(); onContinue(); });
    this.el.choices.appendChild(prompt);
  }

  renderEnding(ending) {
    this._hide(this.el.narrative);
    this._hide(this.el.eventTitle);
    this._hide(this.el.feedback);
    this._hide(this.el.phaseTransition);
    this.el.choices.innerHTML = "";

    this.el.ending.style.display = "";
    const endingClass = ending.type === "survival" ? "success" : ending.type;
    this.el.ending.className = `visible ${endingClass}`;
    this.el.endingTitle.textContent = ending.name;
    this.el.endingText.textContent  = ending.text;
  }

  _show(el) { el.style.display = ""; el.className = ""; }
  _hide(el) { el.style.display = "none"; }

  _setVisible(elObj, keys, visible) {
    for (const k of keys) {
      if (elObj[k]) elObj[k].style.display = visible ? "" : "none";
    }
  }
}
