import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  applyChoice,
  checkEndings,
  createInitialState,
  initGame,
  loadContent,
  runCycle,
  tickSpirit,
  advanceNight,
} from "./yejian_qidao_engine.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const RULES = {
  spirit_cap: 10, spirit_floor: 0,
  suspicion_cap: 10, suspicion_floor: 0,
  faith_cap: 10, faith_floor: 0,
};

function makeConfig(overrides = {}) {
  return {
    nights: {
      total: 5,
      phases: [
        { id: 1, name: "第一夜", events_per_night: 2 },
        { id: 2, name: "第二夜", events_per_night: 2 },
        { id: 3, name: "第三夜", events_per_night: 2 },
        { id: 4, name: "第四夜", events_per_night: 2 },
        { id: 5, name: "第五夜", events_per_night: 2 },
      ],
    },
    initial_state: {
      player: { spirit: 7, suspicion: 0, faith: 5 },
      spirit_drain_per_night: 1,
    },
    thresholds: {
      spirit: { strong: 7, stable: 4, fragile: 2, broken: 0 },
      suspicion: { none: 0, noted: 3, watched: 5, condemned: 8 },
      faith: { doubt: 2, wavering: 4, steady: 6, devout: 8 },
    },
    endings: {
      survivals: [
        {
          id: "faithful_survivor",
          name: "虔诚者",
          condition: "player.spirit > fragile AND player.suspicion < watched AND player.faith >= devout",
          text: "虔诚者结局",
        },
        {
          id: "silent_survivor",
          name: "沉默者",
          condition: "player.spirit > fragile AND player.suspicion < watched AND player.faith <= wavering",
          text: "沉默者结局",
        },
        {
          id: "compromised",
          name: "附和者",
          condition: "player.spirit <= fragile AND player.suspicion < condemned",
          text: "附和者结局",
        },
      ],
      failures: [
        { id: "broken_spirit", name: "精神崩溃", text: "崩溃结局" },
        { id: "condemned", name: "被审判", text: "审判结局" },
      ],
    },
    state_rules: RULES,
    ...overrides,
  };
}

const CHARS = {
  characters: [
    { id: "enoch" },
    { id: "sarah" },
    { id: "nahor" },
    { id: "caleb" },
  ],
};

// ---------------------------------------------------------------------------
// 1. createInitialState
// ---------------------------------------------------------------------------
describe("createInitialState", () => {
  it("initializes player stats from config", () => {
    const state = createInitialState(makeConfig(), CHARS);
    assert.equal(state.currentNight, 1);
    assert.equal(state.currentPhaseId, 1);
    assert.equal(state.eventIndex, 0);
    assert.equal(state.eventsProcessed, 0);
    assert.equal(state.player.spirit, 7);
    assert.equal(state.player.suspicion, 0);
    assert.equal(state.player.faith, 5);
  });

  it("initializes character trust to 0", () => {
    const state = createInitialState(makeConfig(), CHARS);
    assert.equal(state.characters.enoch.trust, 0);
    assert.equal(state.characters.sarah.trust, 0);
    assert.equal(state.characters.nahor.trust, 0);
    assert.equal(state.characters.caleb.trust, 0);
  });

  it("starts with gameOver false and no ending", () => {
    const state = createInitialState(makeConfig(), CHARS);
    assert.equal(state.gameOver, false);
    assert.equal(state.ending, null);
  });
});

// ---------------------------------------------------------------------------
// 2. applyChoice
// ---------------------------------------------------------------------------
describe("applyChoice", () => {
  it("applies player stat effects", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = applyChoice(state, {
      player: { spirit: -2, suspicion: 1, faith: 3 },
    }, RULES);
    assert.equal(next.player.spirit, 5);
    assert.equal(next.player.suspicion, 1);
    assert.equal(next.player.faith, 8);
  });

  it("applies character trust effects", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = applyChoice(state, {
      sarah: { trust: 2 },
      nahor: { trust: 1 },
    }, RULES);
    assert.equal(next.characters.sarah.trust, 2);
    assert.equal(next.characters.nahor.trust, 1);
    assert.equal(next.characters.enoch.trust, 0);
  });

  it("clamps player stats within floor and cap", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = applyChoice(state, {
      player: { spirit: -20, suspicion: 50, faith: -10 },
    }, RULES);
    assert.equal(next.player.spirit, 0);
    assert.equal(next.player.suspicion, 10);
    assert.equal(next.player.faith, 0);
  });

  it("increments eventsProcessed", () => {
    const state = createInitialState(makeConfig(), CHARS);
    assert.equal(state.eventsProcessed, 0);
    const next = applyChoice(state, {}, RULES);
    assert.equal(next.eventsProcessed, 1);
  });

  it("ignores meta and unknown character keys", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = applyChoice(state, {
      meta: { bonus: 5 },
      nonexistent: { trust: 10 },
      player: { spirit: -1 },
    }, RULES);
    assert.equal(next.player.spirit, 6);
    assert.equal(next.characters.sarah.trust, 0);
    assert.ok(!next.characters.nonexistent);
  });

  it("does not mutate original state", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const before = structuredClone(state);
    applyChoice(state, { player: { spirit: -3 }, sarah: { trust: 2 } }, RULES);
    assert.deepEqual(state, before);
  });
});

// ---------------------------------------------------------------------------
// 3. tickSpirit
// ---------------------------------------------------------------------------
describe("tickSpirit", () => {
  it("drains spirit by specified amount", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = tickSpirit(state, 1, RULES);
    assert.equal(next.player.spirit, 6);
  });

  it("drains spirit by default amount of 1", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = tickSpirit(state);
    assert.equal(next.player.spirit, 6);
  });

  it("clamps spirit at floor", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.player.spirit = 0;
    const next = tickSpirit(state, 3, RULES);
    assert.equal(next.player.spirit, 0);
  });

  it("preserves other player stats", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = tickSpirit(state, 1, RULES);
    assert.equal(next.player.suspicion, state.player.suspicion);
    assert.equal(next.player.faith, state.player.faith);
  });
});

// ---------------------------------------------------------------------------
// 4. checkEndings
// ---------------------------------------------------------------------------
describe("checkEndings", () => {
  it("returns null when no condition is met and nights remain", () => {
    const state = createInitialState(makeConfig(), CHARS);
    assert.equal(checkEndings(state, makeConfig()), null);
  });

  it("detects broken_spirit failure when spirit <= 0", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.player.spirit = 0;
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.id, "broken_spirit");
    assert.equal(ending.type, "failure");
  });

  it("detects condemned failure when suspicion >= 8", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.player.suspicion = 8;
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.id, "condemned");
    assert.equal(ending.type, "failure");
  });

  it("failure takes priority over survival", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.currentNight = 6;
    state.player.spirit = 0;
    state.player.faith = 10;
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.type, "failure");
  });

  it("detects faithful_survivor after all nights with high faith", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.currentNight = 6;
    state.player.spirit = 8;
    state.player.suspicion = 2;
    state.player.faith = 9;
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.id, "faithful_survivor");
    assert.equal(ending.type, "survival");
  });

  it("detects silent_survivor with low faith but safe stats", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.currentNight = 6;
    state.player.spirit = 5;
    state.player.suspicion = 1;
    state.player.faith = 3;
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.id, "silent_survivor");
    assert.equal(ending.type, "survival");
  });

  it("detects compromised with fragile spirit", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.currentNight = 6;
    state.player.spirit = 1;
    state.player.suspicion = 3;
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.id, "compromised");
    assert.equal(ending.type, "survival");
  });

  it("returns fallback ending when no survival matches", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.currentNight = 6;
    state.player.spirit = 4;  // > fragile
    state.player.suspicion = 6;  // >= watched — blocks faithful & silent
    state.player.faith = 5;  // not >= devout, not <= wavering
    const ending = checkEndings(state, makeConfig());
    assert.equal(ending.id, "survived_unclassified");
    assert.equal(ending.type, "survival");
  });
});

// ---------------------------------------------------------------------------
// 5. advanceNight
// ---------------------------------------------------------------------------
describe("advanceNight", () => {
  it("advances to next night within total", () => {
    const state = createInitialState(makeConfig(), CHARS);
    const next = advanceNight(state, makeConfig());
    assert.equal(next.currentNight, 2);
    assert.equal(next.currentPhaseId, 2);
    assert.equal(next.eventIndex, 0);
  });

  it("advances past total without updating phase", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.currentNight = 5;
    const next = advanceNight(state, makeConfig());
    assert.equal(next.currentNight, 6);
  });

  it("preserves player and character state", () => {
    const state = createInitialState(makeConfig(), CHARS);
    state.characters.sarah.trust = 5;
    const next = advanceNight(state, makeConfig());
    assert.equal(next.characters.sarah.trust, 5);
    assert.equal(next.player.spirit, state.player.spirit);
  });
});

// ---------------------------------------------------------------------------
// 6. runCycle — full cycle settlement
// ---------------------------------------------------------------------------
describe("runCycle", () => {
  it("first event stays in same night", () => {
    const config = makeConfig();
    const state = createInitialState(config, CHARS);
    const result = runCycle(state, { player: { spirit: -1 } }, config);
    assert.equal(result.ending, null);
    assert.equal(result.state.currentNight, 1);
    assert.equal(result.state.eventsProcessed, 1);
    assert.equal(result.state.player.spirit, 6);
  });

  it("second event advances to next night with spirit drain", () => {
    const config = makeConfig();
    let state = createInitialState(config, CHARS);
    const first = runCycle(state, { player: { spirit: -1 } }, config);
    state = first.state;
    const second = runCycle(state, { player: { spirit: 0 } }, config);
    assert.equal(second.ending, null);
    assert.equal(second.state.currentNight, 2);
    assert.equal(second.state.player.spirit, 5); // 7 - 1 (choice1) = 6, then tick: 6 - 1 = 5
  });

  it("detects failure mid-night", () => {
    const config = makeConfig();
    const state = createInitialState(config, CHARS);
    state.player.spirit = 1;
    const result = runCycle(state, { player: { spirit: -2 } }, config);
    assert.equal(result.ending.id, "broken_spirit");
    assert.equal(result.state.gameOver, true);
  });

  it("does nothing after game over", () => {
    const config = makeConfig();
    const state = createInitialState(config, CHARS);
    state.gameOver = true;
    state.ending = { id: "broken_spirit", type: "failure" };
    const result = runCycle(state, { player: { spirit: 5 } }, config);
    assert.equal(result.state.gameOver, true);
    assert.equal(result.ending.id, "broken_spirit");
  });

  it("settles survival after completing all nights", () => {
    const config = makeConfig();
    let state = createInitialState(config, CHARS);
    // Run 10 events (2 per night × 5 nights) to complete all nights
    for (let i = 0; i < 10 && !state.gameOver; i++) {
      const result = runCycle(state, {
        player: { faith: 1, spirit: 0, suspicion: 0 },
        sarah: { trust: 1 },
      }, config);
      state = result.state;
    }
    assert.equal(state.gameOver, true);
    assert.ok(state.ending);
    assert.ok(state.ending.type === "survival" || state.ending.type === "failure");
  });

  it("completes a faithful survivor run", () => {
    const config = makeConfig();
    let state = createInitialState(config, CHARS);
    // Make faithful choices: +faith, low suspicion
    for (let i = 0; i < 10 && !state.gameOver; i++) {
      const result = runCycle(state, {
        player: { faith: 1, spirit: 1, suspicion: -1 },
      }, config);
      state = result.state;
    }
    assert.equal(state.gameOver, true);
    assert.equal(state.ending.id, "faithful_survivor");
    assert.equal(state.ending.type, "survival");
  });

  it("triggers condemned ending with high suspicion", () => {
    const config = makeConfig();
    const state = createInitialState(config, CHARS);
    state.player.suspicion = 7;
    const result = runCycle(state, { player: { suspicion: 2 } }, config);
    assert.equal(result.ending.id, "condemned");
    assert.equal(result.state.gameOver, true);
  });
});

// ---------------------------------------------------------------------------
// 7. Content loading (requires Node.js)
// ---------------------------------------------------------------------------
describe("content loading", () => {
  it("loads all content files", () => {
    const content = loadContent();
    assert.equal(content.config.game, "yejian-qidao");
    assert.ok(content.characters.characters.length > 0);
    assert.ok(content.events.events.length > 0);
  });

  it("initializes a game from content", () => {
    const game = initGame();
    assert.equal(game.config.game, "yejian-qidao");
    assert.equal(game.state.currentNight, 1);
    assert.equal(game.state.player.spirit, 7);
  });
});
