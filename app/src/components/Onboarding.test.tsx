import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Onboarding } from "./Onboarding";

const base = () => ({
  destination: "local" as const, driveConnected: false, syncRoot: "", cloudGameCount: 0,
  onChooseLocal: vi.fn().mockResolvedValue(undefined), onConnectDrive: vi.fn().mockResolvedValue(undefined), onChooseFolder: vi.fn().mockResolvedValue(undefined),
  onAddGame: vi.fn(), onFinish: vi.fn(),
});

describe("Onboarding", () => {
  it("walks through the steps and opens add-game when finished", () => {
    const props = base();
    render(<Onboarding {...props} />);
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByRole("heading", { name: "Where should saves be kept?" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    fireEvent.click(screen.getByRole("button", { name: "Add a game" }));
    expect(props.onFinish).toHaveBeenCalled();
    expect(props.onAddGame).toHaveBeenCalled();
  });

  it("will not continue with Google Drive until it is connected", () => {
    render(<Onboarding {...base()} destination="google_drive" />);
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByRole("button", { name: "Continue" })).toBeDisabled();
  });

  it("can be skipped on the first step", () => {
    const props = base();
    render(<Onboarding {...props} />);
    fireEvent.click(screen.getByRole("button", { name: "Skip setup" }));
    expect(props.onFinish).toHaveBeenCalled();
  });
});
