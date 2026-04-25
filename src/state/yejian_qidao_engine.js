// 夜间祈祷 (Night Prayer) — pure state engine
// Node-specific imports are lazy so pure functions work in the browser too.

let _readFileSync, _join, CONTENT_DIR;
try {
  const { readFileSync } = await import("node:fs");
  const { dirname, join } = await import("node:path");
  const { fileURLToPath } = await import("node:url");
  const __dirname = dirname(fileURLToPath(import.meta.url));
  CONTENT_DIR = join(__dirname, "..", "content");
  _readFileSync = readFileSync;
  _join = join;
} catch {}

const DEFAULT_RULES = {
  spirit_cap: 10, spirit_floor: 0,
  suspicion_cap: 10, suspicion_floor: 0,
  faith_cap: 10, faith_floor: 0,
};

export function loadContent() {
  return {
    config: readJSON("yejian_qidao_config.json"),
    characters: readJSON("yejian_qidao_characters.json"),
    events: readJSON("yejian_qidao_events.json"),
  };
}

function readJSON(name) {
  if (!_readFileSync || !_join || !CONTENT_DIR) {
    throw new Error("loadContent requires Node.js file APIs");
  }
  return JSON.parse(_readFileSync(_join(CONTENT_DIR, name), "utf-8"));
}

export function createInitialState(config, characters) {
  const phase = config.nights.phases[0];
  const player = config.initial_state.player;
  const characterList = characters.characters ?? characters;
  const characterState = {};

  for (const char of characterList) {
    characterState[char.id] = { trust: 0 };
  }

  return {
    currentNight: 1,
    currentPhaseId: phase.id,
    eventIndex: 0,
    eventsProcessed: 0,
    player: { spirit: player.spirit, suspicion: player.suspicion, faith: player.faith },
    characters: characterState,
    gameOver: false,
    ending: null,
  };
}

export function applyChoice(state, effects = {}, rules = DEFAULT_RULES) {
  const resolvedRules = { ...DEFAULT_RULES, ...rules };
  const characters = { ...state.characters };

  for (const [id, delta] of Object.entries(effects)) {
    if (id === "player" || id === "meta" || !characters[id]) continue;
    const current = characters[id];
    characters[id] = {
      ...current,
      trust: (current.trust ?? 0) + (delta.trust ?? 0),
    };
  }

  const playerEffects = effects.player ?? {};
  const player = clampPlayer({
    ...state.player,
    spirit: state.player.spirit + (playerEffects.spirit ?? 0),
    suspicion: state.player.suspicion + (playerEffects.suspicion ?? 0),
    faith: state.player.faith + (playerEffects.faith ?? 0),
  }, resolvedRules);

  return {
    ...state,
    player,
    characters,
    eventsProcessed: state.eventsProcessed + 1,
  };
}

export function tickSpirit(state, drain = 1, rules = DEFAULT_RULES) {
  const resolvedRules = { ...DEFAULT_RULES, ...rules };
  return {
    ...state,
    player: clampPlayer({
      ...state.player,
      spirit: state.player.spirit - drain,
    }, resolvedRules),
  };
}

export function checkEndings(state, config) {
  const { thresholds, endings } = config;

  if (state.player.spirit <= thresholds.spirit.broken) {
    return { ...endings.failures.find((f) => f.id === "broken_spirit"), type: "failure" };
  }
  if (state.player.suspicion >= thresholds.suspicion.condemned) {
    return { ...endings.failures.find((f) => f.id === "condemned"), type: "failure" };
  }

  if (state.currentNight <= config.nights.total) return null;

  for (const survival of endings.survivals) {
    if (checkCondition(survival.condition, state.player, thresholds)) {
      return { ...survival, type: "survival" };
    }
  }

  return {
    id: "survived_unclassified",
    name: "幸存者",
    type: "survival",
    text: "五夜过去了。你跪在那里，说不清自己是虔诚还是麻木。以诺没有再看你。你活了下来——至于灵魂里还剩什么，只有黑夜知道。",
  };
}

function checkCondition(condition, player, thresholds) {
  const parts = condition.split(" AND ");
  for (const part of parts) {
    const match = part.trim().match(/player\.(\w+)\s*([><=!]+)\s*(\w+)/);
    if (!match) continue;
    const [, stat, op, thresholdName] = match;
    const value = player[stat];
    const threshold = thresholds[stat]?.[thresholdName];
    if (threshold === undefined) continue;

    if (op === ">" && !(value > threshold)) return false;
    if (op === ">=" && !(value >= threshold)) return false;
    if (op === "<" && !(value < threshold)) return false;
    if (op === "<=" && !(value <= threshold)) return false;
  }
  return true;
}

export function advanceNight(state, config) {
  const nextNight = state.currentNight + 1;
  if (nextNight > config.nights.total) {
    return { ...state, currentNight: nextNight };
  }
  const phase = config.nights.phases.find((p) => p.id === nextNight);
  return {
    ...state,
    currentNight: nextNight,
    currentPhaseId: phase.id,
    eventIndex: 0,
  };
}

export function runCycle(state, choiceEffects, config) {
  if (state.gameOver) {
    return { state, ending: state.ending };
  }

  const phase = config.nights.phases.find((p) => p.id === state.currentPhaseId);
  const eventsPerNight = phase?.events_per_night ?? 2;
  let nextState = applyChoice(state, choiceEffects, config.state_rules);

  if (nextState.eventsProcessed % eventsPerNight !== 0) {
    return settleIfNeeded(nextState, config);
  }

  nextState = tickSpirit(nextState, config.initial_state.spirit_drain_per_night, config.state_rules);

  const nightEnding = checkEndings(nextState, config);
  if (nightEnding) return finish(nextState, nightEnding);

  nextState = advanceNight(nextState, config);

  const finalEnding = checkEndings(nextState, config);
  if (finalEnding) return finish(nextState, finalEnding);

  return { state: nextState, ending: null };
}

export function initGame() {
  const content = loadContent();
  return {
    content,
    config: content.config,
    state: createInitialState(content.config, content.characters),
  };
}

function settleIfNeeded(state, config) {
  const ending = checkEndings(state, config);
  if (ending) return finish(state, ending);
  return { state, ending: null };
}

function finish(state, ending) {
  return { state: { ...state, gameOver: true, ending }, ending };
}

function clampPlayer(player, rules) {
  return {
    ...player,
    spirit: clamp(player.spirit, rules.spirit_floor, rules.spirit_cap),
    suspicion: clamp(player.suspicion, rules.suspicion_floor, rules.suspicion_cap),
    faith: clamp(player.faith, rules.faith_floor, rules.faith_cap),
  };
}

function clamp(value, floor, cap) {
  return Math.max(floor, Math.min(cap, value));
}
