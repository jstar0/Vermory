import { describe, expect, it } from "vitest";

import { normalizePluginConfig } from "../src/config.js";

describe("normalizePluginConfig", () => {
  it("uses bounded loopback defaults", () => {
    expect(normalizePluginConfig(undefined)).toEqual({
      enabled: true,
      baseUrl: "http://127.0.0.1:8787",
      timeoutMs: 5000,
    });
  });

  it("normalizes a configured HTTP endpoint", () => {
    expect(
      normalizePluginConfig({
        enabled: false,
        baseUrl: " https://vermory.example.test/api/ ",
        timeoutMs: 12000,
      }),
    ).toEqual({
      enabled: false,
      baseUrl: "https://vermory.example.test/api",
      timeoutMs: 12000,
    });
  });

  it.each([
    ["non-object", "invalid"],
    ["non-http URL", { baseUrl: "file:///tmp/vermory" }],
    ["credential-bearing URL", { baseUrl: "http://user:secret@127.0.0.1:8787" }],
    ["short timeout", { timeoutMs: 249 }],
    ["long timeout", { timeoutMs: 30001 }],
    ["request-owned tenant", { tenantId: "attacker" }],
    ["request-owned continuity", { continuityId: "attacker" }],
  ])("rejects %s", (_name, input) => {
    expect(() => normalizePluginConfig(input)).toThrow();
  });
});
