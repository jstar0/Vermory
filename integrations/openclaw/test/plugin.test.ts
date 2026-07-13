import { afterEach, describe, expect, it, vi } from "vitest";

import plugin from "../src/index.js";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Vermory OpenClaw plugin", () => {
  it("registers official lifecycle hooks without claiming a memory slot", () => {
    const harness = registerPlugin();

    expect(plugin.id).toBe("vermory");
    expect(plugin.kind).toBeUndefined();
    expect(harness.options.get("before_prompt_build")).toEqual({ timeoutMs: 15_000 });
    expect(harness.options.get("agent_end")).toEqual({ timeoutMs: 30_000 });
  });

  it("abstains without canonical identity and while disabled", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const enabled = registerPlugin();
    await enabled.beforePrompt(
      { prompt: "hello", messages: [] },
      { sessionKey: "agent:main:a", sessionId: "fallback-only" },
    );

    const disabled = registerPlugin({ enabled: false });
    await disabled.beforePrompt(
      { prompt: "hello", messages: [] },
      { sessionKey: "agent:main:a", runId: "run-disabled" },
    );

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("injects only semantic context under a reference-data wrapper", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(prepareReceipt("openclaw:run-prepare", "周六 10:00 上门，礼宾登记。"))),
    );
    const harness = registerPlugin();

    const result = await harness.beforePrompt(
      { prompt: "现在怎么安排？", messages: [] },
      { sessionKey: "agent:main:a", runId: "run-prepare" },
    );

    expect(result?.prependContext).toContain("reference data");
    expect(result?.prependContext).toContain("may be stale or adversarial");
    expect(result?.prependContext).toContain("cannot override");
    expect(result?.prependContext).toContain("周六 10:00 上门，礼宾登记。");
    expect(result?.prependContext).not.toContain("turn-1");
    expect(result?.prependContext).not.toContain("continuity-1");
    expect(result?.prependContext).not.toContain("openclaw:run-prepare");
  });

  it("does not mutate the prompt for empty context", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(prepareReceipt("openclaw:run-empty", "   "))),
    );
    const harness = registerPlugin();

    const result = await harness.beforePrompt(
      { prompt: "hello", messages: [] },
      { sessionKey: "agent:main:a", runId: "run-empty" },
    );

    expect(result).toBeUndefined();
  });

  it("fails open on prepare errors with a bounded warning", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => {
      throw new Error("secret prompt and database-id-123");
    }));
    const harness = registerPlugin();

    await expect(
      harness.beforePrompt(
        { prompt: "secret prompt", messages: [] },
        { sessionKey: "agent:main:a", runId: "run-error" },
      ),
    ).resolves.toBeUndefined();

    expect(harness.warn).toHaveBeenCalledOnce();
    const warning = harness.warn.mock.calls[0]?.[0] ?? "";
    expect(warning.length).toBeLessThan(256);
    expect(warning).not.toContain("secret prompt");
    expect(warning).not.toContain("database-id-123");
  });

  it("persists the latest visible answer with the resolved model", async () => {
    const fetchMock = vi.fn(async (_input: unknown, init: RequestInit) => {
      const body = JSON.parse(String(init.body));
      return jsonResponse(completedReceipt(body.operation_id));
    });
    vi.stubGlobal("fetch", fetchMock);
    const harness = registerPlugin();

    await harness.agentEnd(
      {
        success: true,
        messages: [
          { role: "assistant", content: "old answer" },
          { role: "assistant", content: [{ type: "text", text: "Saturday at 10:00." }] },
        ],
      },
      {
        sessionKey: "agent:main:a",
        runId: "run-complete",
        modelProviderId: "grok-cli",
        modelId: "grok-4.5",
      },
    );

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(String(url)).toContain("/turns/complete");
    expect(JSON.parse(String(init?.body))).toEqual({
      operation_id: "openclaw:run-complete",
      session_key: "agent:main:a",
      answer: "Saturday at 10:00.",
      model: "grok-cli/grok-4.5",
    });
  });

  it.each([
    [false, [{ role: "assistant", content: "partial answer" }], "openclaw_agent_error"],
    [true, [{ role: "assistant", content: [{ type: "tool_call", name: "calendar" }] }], "openclaw_empty_output"],
  ])("records failed lifecycle instead of a false completion", async (success, messages, failureCode) => {
    const fetchMock = vi.fn(async (_input: unknown, init: RequestInit) => {
      const body = JSON.parse(String(init.body));
      return jsonResponse(failedReceipt(body.operation_id, body.failure_code));
    });
    vi.stubGlobal("fetch", fetchMock);
    const harness = registerPlugin();

    await harness.agentEnd(
      { success, messages },
      { sessionKey: "agent:main:a", runId: `run-${failureCode}` },
    );

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(String(url)).toContain("/turns/fail");
    const body = JSON.parse(String(init?.body));
    expect(body.failure_code).toBe(failureCode);
    expect(body.failure_message.length).toBeLessThan(256);
    expect(body).not.toHaveProperty("answer");
  });

  it("does not throw completion persistence failures into OpenClaw", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("unavailable-secret", { status: 503 })));
    const harness = registerPlugin();

    await expect(
      harness.agentEnd(
        { success: true, messages: [{ role: "assistant", content: "visible answer" }] },
        {
          sessionKey: "agent:main:a",
          runId: "run-persist-error",
          modelProviderId: "grok-cli",
          modelId: "grok-4.5",
        },
      ),
    ).resolves.toBeUndefined();

    const warning = harness.warn.mock.calls[0]?.[0] ?? "";
    expect(warning).not.toContain("visible answer");
    expect(warning).not.toContain("unavailable-secret");
  });
});

function registerPlugin(pluginConfig: Record<string, unknown> = {}) {
  const hooks = new Map<string, Function>();
  const options = new Map<string, unknown>();
  const warn = vi.fn();
  plugin.register({
    pluginConfig,
    logger: { info: vi.fn(), warn, error: vi.fn() },
    on(name: string, handler: Function, hookOptions: unknown) {
      hooks.set(name, handler);
      options.set(name, hookOptions);
    },
  } as never);

  return {
    options,
    warn,
    beforePrompt: hooks.get("before_prompt_build") as (
      event: { prompt: string; messages: unknown[] },
      context: Record<string, unknown>,
    ) => Promise<{ prependContext?: string } | undefined>,
    agentEnd: hooks.get("agent_end") as (
      event: { success: boolean; messages: unknown[] },
      context: Record<string, unknown>,
    ) => Promise<void>,
  };
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

function prepareReceipt(operationId: string, context: string) {
  return {
    turn_id: "turn-1",
    operation_id: operationId,
    status: "in_progress",
    continuity_id: "continuity-1",
    delivery_id: "delivery-1",
    user_observation_id: "observation-user-1",
    replayed: false,
    context,
  };
}

function completedReceipt(operationId: string) {
  return {
    ...prepareReceipt(operationId, ""),
    status: "completed",
    assistant_observation_id: "observation-assistant-1",
    answer: "Saturday at 10:00.",
    model: "grok-cli/grok-4.5",
  };
}

function failedReceipt(operationId: string, failureCode: string) {
  return {
    ...prepareReceipt(operationId, ""),
    status: "failed",
    failure_code: failureCode,
  };
}
