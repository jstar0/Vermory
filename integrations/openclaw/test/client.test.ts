import { createServer } from "node:http";
import type { AddressInfo } from "node:net";

import { describe, expect, it } from "vitest";

import { VermoryClient } from "../src/client.js";

const TEST_API_TOKEN =
  "vmt_0123456789abcdef01234567_MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY";

describe("VermoryClient", () => {
  it("posts an exact prepare request and validates the receipt", async () => {
    await withServer(async (request, response) => {
      expect(request.method).toBe("POST");
      expect(request.url).toBe("/v1/integrations/openclaw/turns/prepare");
      expect(request.headers["content-type"]).toBe("application/json");
      expect(request.headers.authorization).toBeUndefined();
      expect(JSON.parse(await readRequest(request))).toEqual({
        operation_id: "openclaw:run-1",
        session_key: "agent:main:a",
        message: "What is the current appointment?",
      });
      writeJSON(response, prepareReceipt("openclaw:run-1", "Use Saturday at 10:00."));
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
      const receipt = await client.prepare({
        operationId: "openclaw:run-1",
        sessionKey: "agent:main:a",
        message: "What is the current appointment?",
      });

      expect(receipt.context).toBe("Use Saturday at 10:00.");
      expect(receipt.status).toBe("in_progress");
    });
  });

  it("uses each server-filtered context without retaining an earlier eligible fact", async () => {
    let prepareCount = 0;
    await withServer(async (request, response) => {
      const body = JSON.parse(await readRequest(request));
      prepareCount += 1;
      if (prepareCount === 1) {
        writeJSON(
          response,
          prepareReceipt(body.operation_id, "Temporary workaround: set VERMORY_CACHE_DISABLED=1.\nRun go test -p 1 -count=1 ./..."),
        );
        return;
      }
      writeJSON(response, prepareReceipt(body.operation_id, "Run go test -p 1 -count=1 ./..."));
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
      const before = await client.prepare({
        operationId: "openclaw:eligibility-before",
        sessionKey: "agent:main:w03",
        message: "How should verification run?",
      });
      const after = await client.prepare({
        operationId: "openclaw:eligibility-after",
        sessionKey: "agent:main:w03",
        message: "How should verification run now?",
      });

      expect(before.context).toContain("VERMORY_CACHE_DISABLED=1");
      expect(after.context).toBe("Run go test -p 1 -count=1 ./...");
      expect(after.context).not.toContain("VERMORY_CACHE_DISABLED=1");
      expect(prepareCount).toBe(2);
    });
  });

  it("adds one exact bearer header without exposing it elsewhere", async () => {
    await withServer(async (request, response) => {
      expect(request.headers.authorization).toBe(`Bearer ${TEST_API_TOKEN}`);
      writeJSON(response, prepareReceipt("openclaw:authenticated", "governed context"));
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 1000, apiToken: `  ${TEST_API_TOKEN}\n` });
      await client.prepare({
        operationId: "openclaw:authenticated",
        sessionKey: "agent:main:a",
        message: "hello",
      });
    });
  });

  it.each([
    `vmt_0123456789abcdef01234567_bad secret`,
    `vmt_0123456789abcdef01234567_bad\u0000secret`,
    "not-a-vermory-token",
  ])("rejects malformed bearer material without echoing it", (apiToken) => {
    let message = "";
    try {
      new VermoryClient({ baseUrl: "http://127.0.0.1:8787", timeoutMs: 1000, apiToken });
    } catch (error) {
      message = error instanceof Error ? error.message : String(error);
    }
    expect(message).toContain("Vermory API token is invalid");
    expect(message).not.toContain(apiToken);
  });

  it("posts exact complete and fail requests", async () => {
    const requests: Array<{ path: string; body: unknown }> = [];
    await withServer(async (request, response) => {
      const body = JSON.parse(await readRequest(request));
      requests.push({ path: request.url ?? "", body });
      if (request.url?.endsWith("/complete")) {
        writeJSON(response, completedReceipt(body.operation_id));
        return;
      }
      writeJSON(response, failedReceipt(body.operation_id, body.failure_code));
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
      await client.complete({
        operationId: "openclaw:run-complete",
        sessionKey: "agent:main:a",
        answer: "The appointment is Saturday at 10:00.",
        model: "grok-cli/grok-4.5",
      });
      await client.fail({
        operationId: "openclaw:run-fail",
        sessionKey: "agent:main:a",
        failureCode: "openclaw_agent_error",
        failureMessage: "no visible answer",
      });
    });

    expect(requests).toEqual([
      {
        path: "/v1/integrations/openclaw/turns/complete",
        body: {
          operation_id: "openclaw:run-complete",
          session_key: "agent:main:a",
          answer: "The appointment is Saturday at 10:00.",
          model: "grok-cli/grok-4.5",
        },
      },
      {
        path: "/v1/integrations/openclaw/turns/fail",
        body: {
          operation_id: "openclaw:run-fail",
          session_key: "agent:main:a",
          failure_code: "openclaw_agent_error",
          failure_message: "no visible answer",
        },
      },
    ]);
  });

  it("aborts requests at the configured timeout", async () => {
    await withServer((_request, response) => {
      setTimeout(() => writeJSON(response, prepareReceipt("openclaw:slow", "late")), 500);
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 250 });
      await expect(
        client.prepare({
          operationId: "openclaw:slow",
          sessionKey: "agent:main:a",
          message: "secret prompt",
        }),
      ).rejects.toThrow("Vermory prepare failed");
    });
  });

  it("does not expose request or response bodies in HTTP errors", async () => {
    await withServer((_request, response) => {
      response.writeHead(500, { "content-type": "application/json" });
      response.end('{"error":"secret prompt and database-id-123"}');
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
      let message = "";
      try {
        await client.prepare({
          operationId: "openclaw:http-error",
          sessionKey: "agent:main:a",
          message: "secret prompt",
        });
      } catch (error) {
        message = error instanceof Error ? error.message : String(error);
      }

      expect(message).toContain("HTTP 500");
      expect(message).not.toContain("secret prompt");
      expect(message).not.toContain("database-id-123");
    });
  });

  it("rejects an oversized response before JSON parsing", async () => {
    await withServer((_request, response) => {
      writeJSON(response, {
        ...prepareReceipt("openclaw:oversized", "ok"),
        padding: "x".repeat(300 * 1024),
      });
    }, async (baseUrl) => {
      const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
      await expect(
        client.prepare({
          operationId: "openclaw:oversized",
          sessionKey: "agent:main:a",
          message: "hello",
        }),
      ).rejects.toThrow("Vermory prepare response is too large");
    });
  });

  it("rejects invalid JSON and malformed receipts without leaking content", async () => {
    const bodies = [
      "not-json-secret",
      JSON.stringify({ ...prepareReceipt("openclaw:wrong", "context"), turn_id: "" }),
    ];

    for (const body of bodies) {
      await withServer((_request, response) => {
        response.writeHead(200, { "content-type": "application/json" });
        response.end(body);
      }, async (baseUrl) => {
        const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
        let message = "";
        try {
          await client.prepare({
            operationId: "openclaw:wrong",
            sessionKey: "agent:main:a",
            message: "hello",
          });
        } catch (error) {
          message = error instanceof Error ? error.message : String(error);
        }
        expect(message).toContain("Vermory prepare");
        expect(message).not.toContain("not-json-secret");
        expect(message).not.toContain("context");
      });
    }
  });

  it("rejects a receipt for another operation or lifecycle phase", async () => {
    for (const receipt of [
      prepareReceipt("openclaw:other", "context"),
      { ...prepareReceipt("openclaw:expected", "context"), status: "completed" },
    ]) {
      await withServer((_request, response) => writeJSON(response, receipt), async (baseUrl) => {
        const client = new VermoryClient({ baseUrl, timeoutMs: 1000 });
        await expect(
          client.prepare({
            operationId: "openclaw:expected",
            sessionKey: "agent:main:a",
            message: "hello",
          }),
        ).rejects.toThrow("Vermory prepare returned an invalid receipt");
      });
    }
  });
});

async function withServer(
  handler: Parameters<typeof createServer>[0],
  run: (baseUrl: string) => Promise<void>,
): Promise<void> {
  const server = createServer(handler);
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address() as AddressInfo;
  try {
    await run(`http://127.0.0.1:${address.port}`);
  } finally {
    await new Promise<void>((resolve, reject) => {
      server.close((error) => (error ? reject(error) : resolve()));
    });
  }
}

async function readRequest(request: AsyncIterable<Uint8Array>): Promise<string> {
  const chunks: Uint8Array[] = [];
  for await (const chunk of request) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString("utf8");
}

function writeJSON(response: { writeHead: Function; end: Function }, value: unknown): void {
  response.writeHead(200, { "content-type": "application/json" });
  response.end(JSON.stringify(value));
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
    answer: "The appointment is Saturday at 10:00.",
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
