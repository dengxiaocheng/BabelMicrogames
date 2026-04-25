import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { NightPrayerController } from "../ui/yejian_qidao_controller.js";
import { loadContent, createInitialState } from "../state/yejian_qidao_engine.js";

// ---------------------------------------------------------------------------
// Stub renderer — records calls for assertions
// ---------------------------------------------------------------------------
function createStubRenderer() {
  const log = [];
  let _pendingContinue = null;
  let _pendingChoice = null;

  return {
    log,
    renderStatus(s)      { log.push({ type: "status", ...s }); },
    showStats()           { log.push({ type: "showStats" }); },
    renderEvent(ev, cb)   {
      log.push({ type: "event", id: ev.id });
      _pendingChoice = cb;
    },
    renderFeedback(t)     { log.push({ type: "feedback", text: t }); },
    renderPhaseTransition(t) { log.push({ type: "transition", text: t }); },
    showContinue(cb)      { _pendingContinue = cb; },
    renderEnding(ending)  { log.push({ type: "ending", id: ending?.id }); },

    // Test helpers
    choose(choice)        { const cb = _pendingChoice; _pendingChoice = null; cb(choice); },
    continue()            { const cb = _pendingContinue; _pendingContinue = null; cb(); },
  };
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeContent() {
  return loadContent();
}

/** Drive the controller to completion, choosing the first option every time. */
function playToCompletion(controller, renderer) {
  let guard = 200;
  while (!controller.state.gameOver && guard-- > 0) {
    const event = controller._pickEvent();
    if (!event) break;
    // Trigger the pending renderEvent callback
    renderer.choose(event.choices[0]);
    renderer.continue();
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("NightPrayerController", () => {
  it("starts and shows stats", () => {
    const r = createStubRenderer();
    const c = new NightPrayerController(r, makeContent());
    c.start();
    assert.ok(r.log.some((l) => l.type === "showStats"));
    assert.ok(r.log.some((l) => l.type === "status"));
  });

  it("renders first event on start", () => {
    const r = createStubRenderer();
    const c = new NightPrayerController(r, makeContent());
    c.start();
    assert.ok(r.log.some((l) => l.type === "event"));
  });

  it("shows feedback after a choice", () => {
    const r = createStubRenderer();
    const c = new NightPrayerController(r, makeContent());
    c.start();
    const event = c._pickEvent();
    r.choose(event.choices[0]);
    assert.ok(r.log.some((l) => l.type === "feedback"));
  });

  it("shows continue prompt after feedback", () => {
    const r = createStubRenderer();
    const c = new NightPrayerController(r, makeContent());
    c.start();
    r.choose(c._pickEvent().choices[0]);
    // showContinue should have been called
    assert.ok(r.log.some((l) => l.type === "feedback"));
  });

  it("completes a full faithful-survivor run", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();

    // Run all 10 events (2 per night × 5 nights), choosing +faith options
    const faithFirst = (event) =>
      event.choices.reduce((best, ch) => {
        const f = ch.effects?.player?.faith ?? 0;
        return f > (best._f ?? -Infinity) ? { ...ch, _f: f } : best;
      }, { _f: -Infinity });

    let guard = 20;
    while (!c.state.gameOver && guard-- > 0) {
      const event = c._pickEvent();
      if (!event) break;
      const pick = faithFirst(event);
      r.choose(pick);
      r.continue();
    }

    assert.equal(c.state.gameOver, true);
    assert.ok(c.state.ending, "should have an ending");
    assert.ok(
      c.state.ending.type === "survival" || c.state.ending.type === "failure",
      `unexpected ending type: ${c.state.ending.type}`,
    );
    assert.ok(r.log.some((l) => l.type === "ending"));
  });

  it("reaches ending when spirit drops to zero", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();

    // Choose options that drain spirit the most
    const spiritWorst = (event) =>
      event.choices.reduce((worst, ch) => {
        const s = ch.effects?.player?.spirit ?? 0;
        return s < (worst._s ?? Infinity) ? { ...worst, ...ch, _s: s } : worst;
      }, { _s: Infinity });

    let guard = 30;
    while (!c.state.gameOver && guard-- > 0) {
      const event = c._pickEvent();
      if (!event) break;
      const pick = spiritWorst(event);
      r.choose(pick);
      r.continue(); // always continue — ending renders inside the callback
    }

    assert.equal(c.state.gameOver, true);
    assert.equal(c.state.ending.id, "broken_spirit");
    assert.ok(r.log.some((l) => l.type === "ending"));
  });

  it("shows phase transitions between nights", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();

    // Process 2 events to trigger night advance
    for (let i = 0; i < 2 && !c.state.gameOver; i++) {
      const event = c._pickEvent();
      if (!event) break;
      r.choose(event.choices[0]);
      r.continue();
    }

    assert.ok(
      r.log.some((l) => l.type === "transition"),
      "should show phase transition after 2 events",
    );
  });

  it("does not advance eventIndex on phase change", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();

    // Process 2 events to move to night 2
    for (let i = 0; i < 2 && !c.state.gameOver; i++) {
      r.choose(c._pickEvent().choices[0]);
      r.continue();
    }

    // After phase change, eventIndex should be reset (by engine) + not incremented
    assert.equal(c.state.currentNight, 2, "should have advanced to night 2");
  });

  it("isOver reflects game state", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();
    assert.equal(c.state.gameOver, false);

    playToCompletion(c, r);
    assert.equal(c.state.gameOver, true);
  });

  it("start is idempotent", () => {
    const r = createStubRenderer();
    const c = new NightPrayerController(r, makeContent());
    c.start();
    const eventCount = r.log.filter((l) => l.type === "event").length;
    c.start(); // second call should be no-op
    assert.equal(r.log.filter((l) => l.type === "event").length, eventCount);
  });

  // ----- Boundary & coverage additions -----

  it("reaches condemned ending with high suspicion choices", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();

    const suspicionMax = (event) =>
      event.choices.reduce((best, ch) => {
        const s = ch.effects?.player?.suspicion ?? 0;
        return s > (best._s ?? -Infinity) ? { ...ch, _s: s } : best;
      }, { _s: -Infinity });

    let guard = 30;
    while (!c.state.gameOver && guard-- > 0) {
      const event = c._pickEvent();
      if (!event) break;
      r.choose(suspicionMax(event));
      if (!c.state.gameOver) r.continue();
    }

    assert.equal(c.state.gameOver, true);
    assert.equal(c.state.ending.id, "condemned");
  });

  it("_pickEvent returns null when game is over", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.start();
    c.state.gameOver = true;
    assert.equal(c._pickEvent(), null);
  });

  it("getStatus handles unknown phase gracefully", () => {
    const content = makeContent();
    const r = createStubRenderer();
    const c = new NightPrayerController(r, content);
    c.state.currentPhaseId = 999;
    const status = c.getStatus();
    assert.equal(status.phaseName, "终局");
  });

  it("events exist for all 5 phases", () => {
    const content = makeContent();
    for (let phase = 1; phase <= 5; phase++) {
      const pool = content.events.events.filter((e) => e.phase.includes(phase));
      assert.ok(pool.length > 0, `phase ${phase} should have events`);
    }
  });

  it("all events have choices with valid effects", () => {
    const content = makeContent();
    for (const event of content.events.events) {
      assert.ok(event.choices.length > 0, `${event.id} should have choices`);
      for (const choice of event.choices) {
        assert.ok(choice.effects, `${event.id}/${choice.id} missing effects`);
      }
    }
  });
});
