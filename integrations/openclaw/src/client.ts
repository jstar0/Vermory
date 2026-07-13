const MAX_RESPONSE_BYTES = 256 * 1024;

type TurnStatus = "in_progress" | "completed" | "failed";

export interface ClientConfig {
  baseUrl: string;
  timeoutMs: number;
}

export interface TurnIdentityInput {
  operationId: string;
  sessionKey: string;
}

export interface PreparedTurnRequest extends TurnIdentityInput {
  message: string;
}

export interface CompleteTurnRequest extends TurnIdentityInput {
  answer: string;
  model: string;
}

export interface FailTurnRequest extends TurnIdentityInput {
  failureCode: string;
  failureMessage: string;
}

export interface TurnReceipt {
  turnId: string;
  operationId: string;
  status: TurnStatus;
  continuityId: string;
  deliveryId: string;
  userObservationId: string;
  assistantObservationId?: string;
  answer?: string;
  model?: string;
  failureCode?: string;
  replayed: boolean;
}

export interface PreparedTurnReceipt extends TurnReceipt {
  context?: string;
}

export class VermoryClient {
  constructor(private readonly config: ClientConfig) {}

  async prepare(request: PreparedTurnRequest): Promise<PreparedTurnReceipt> {
    const body = await this.post("prepare", {
      operation_id: request.operationId,
      session_key: request.sessionKey,
      message: request.message,
    });
    return parseReceipt(body, "prepare", request.operationId, true);
  }

  async complete(request: CompleteTurnRequest): Promise<TurnReceipt> {
    const body = await this.post("complete", {
      operation_id: request.operationId,
      session_key: request.sessionKey,
      answer: request.answer,
      model: request.model,
    });
    return parseReceipt(body, "complete", request.operationId, false);
  }

  async fail(request: FailTurnRequest): Promise<TurnReceipt> {
    const body = await this.post("fail", {
      operation_id: request.operationId,
      session_key: request.sessionKey,
      failure_code: request.failureCode,
      failure_message: request.failureMessage,
    });
    return parseReceipt(body, "fail", request.operationId, false);
  }

  private async post(phase: "prepare" | "complete" | "fail", body: unknown): Promise<unknown> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.config.timeoutMs);
    const url = `${this.config.baseUrl}/v1/integrations/openclaw/turns/${phase}`;

    try {
      let response: Response;
      try {
        response = await fetch(url, {
          method: "POST",
          headers: { "content-type": "application/json" },
          body: JSON.stringify(body),
          signal: controller.signal,
        });
      } catch (error) {
        if (controller.signal.aborted) {
          throw new Error(`Vermory ${phase} failed: timeout`);
        }
        throw new Error(`Vermory ${phase} failed: network error`);
      }

      if (!response.ok) {
        throw new Error(`Vermory ${phase} failed: HTTP ${response.status}`);
      }

      const text = await readBoundedResponse(response, phase);
      try {
        return JSON.parse(text) as unknown;
      } catch {
        throw new Error(`Vermory ${phase} returned invalid JSON`);
      }
    } catch (error) {
      if (error instanceof Error && error.message.startsWith("Vermory ")) {
        throw error;
      }
      throw new Error(`Vermory ${phase} failed`);
    } finally {
      clearTimeout(timeout);
    }
  }
}

async function readBoundedResponse(response: Response, phase: string): Promise<string> {
  const contentLength = response.headers.get("content-length");
  if (contentLength !== null && Number(contentLength) > MAX_RESPONSE_BYTES) {
    throw new Error(`Vermory ${phase} response is too large`);
  }

  if (!response.body) {
    const text = await response.text();
    if (new TextEncoder().encode(text).byteLength > MAX_RESPONSE_BYTES) {
      throw new Error(`Vermory ${phase} response is too large`);
    }
    return text;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let total = 0;
  let text = "";
  try {
    while (true) {
      const result = await reader.read();
      if (result.done) {
        text += decoder.decode();
        return text;
      }
      total += result.value.byteLength;
      if (total > MAX_RESPONSE_BYTES) {
        await reader.cancel();
        throw new Error(`Vermory ${phase} response is too large`);
      }
      text += decoder.decode(result.value, { stream: true });
    }
  } finally {
    reader.releaseLock();
  }
}

function parseReceipt(
  value: unknown,
  phase: "prepare" | "complete" | "fail",
  operationId: string,
  includeContext: boolean,
): PreparedTurnReceipt | TurnReceipt {
  if (!isRecord(value)) {
    throw new Error(`Vermory ${phase} returned an invalid receipt`);
  }

  const expectedStatus: TurnStatus =
    phase === "prepare" ? "in_progress" : phase === "complete" ? "completed" : "failed";
  if (
    value.operation_id !== operationId ||
    value.status !== expectedStatus ||
    typeof value.turn_id !== "string" ||
    value.turn_id === "" ||
    typeof value.continuity_id !== "string" ||
    value.continuity_id === "" ||
    typeof value.delivery_id !== "string" ||
    value.delivery_id === "" ||
    typeof value.user_observation_id !== "string" ||
    value.user_observation_id === "" ||
    typeof value.replayed !== "boolean"
  ) {
    throw new Error(`Vermory ${phase} returned an invalid receipt`);
  }

  if (phase === "complete" &&
      (typeof value.assistant_observation_id !== "string" ||
        value.assistant_observation_id === "" ||
        typeof value.answer !== "string" ||
        typeof value.model !== "string")) {
    throw new Error(`Vermory ${phase} returned an invalid receipt`);
  }
  if (phase === "fail" &&
      (typeof value.failure_code !== "string" || value.failure_code === "")) {
    throw new Error(`Vermory ${phase} returned an invalid receipt`);
  }
  if (includeContext && value.context !== undefined && typeof value.context !== "string") {
    throw new Error(`Vermory ${phase} returned an invalid receipt`);
  }

  const receipt: TurnReceipt = {
    turnId: value.turn_id,
    operationId: value.operation_id,
    status: expectedStatus,
    continuityId: value.continuity_id,
    deliveryId: value.delivery_id,
    userObservationId: value.user_observation_id,
    replayed: value.replayed,
  };
  if (typeof value.assistant_observation_id === "string") {
    receipt.assistantObservationId = value.assistant_observation_id;
  }
  if (typeof value.answer === "string") {
    receipt.answer = value.answer;
  }
  if (typeof value.model === "string") {
    receipt.model = value.model;
  }
  if (typeof value.failure_code === "string") {
    receipt.failureCode = value.failure_code;
  }
  if (includeContext) {
    return {
      ...receipt,
      ...(typeof value.context === "string" ? { context: value.context } : {}),
    };
  }
  return receipt;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
