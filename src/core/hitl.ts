import { EventEmitter } from "events";

export class HitLPauseError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "HitLPauseError";
  }
}

class HumanInTheLoopManager extends EventEmitter {
  private pendingRequestId: string | null = null;
  public pendingAnswer: string | null = null;

  public requestApproval(question: string): string {
    if (this.pendingRequestId) {
      return "[Error] Another approval request is already pending.";
    }

    this.pendingRequestId = Date.now().toString();
    this.pendingAnswer = null;
    this.emit("approval_requested", { question, requestId: this.pendingRequestId });

    // Return the specific string that the Orchestrator will intercept
    return `[HitL_SUSPENDED]`;
  }

  public hasPendingRequest(): boolean {
    return this.pendingRequestId !== null;
  }

  public resolveRequest(answer: string): void {
    if (this.pendingRequestId) {
      this.pendingAnswer = answer;
      this.pendingRequestId = null;
    }
  }

  public consumeAnswer(): string | null {
    const ans = this.pendingAnswer;
    this.pendingAnswer = null;
    return ans;
  }
}

export const HitL = new HumanInTheLoopManager();
