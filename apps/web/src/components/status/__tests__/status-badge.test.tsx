import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CIStatusBadge, ReviewBadge, DiffStats, StackStatusBadge } from "../status-badge";

describe("CIStatusBadge", () => {
  it("renders nothing when ci is undefined", () => {
    const { container } = render(<CIStatusBadge />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when status is none", () => {
    const { container } = render(
      <CIStatusBadge ci={{ status: "none", reviewDecision: "" }} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders 'CI passing' for passing status", () => {
    render(<CIStatusBadge ci={{ status: "passing", reviewDecision: "" }} />);
    expect(screen.getByText("CI passing")).toBeInTheDocument();
  });

  it("renders 'CI pending' for pending status", () => {
    render(<CIStatusBadge ci={{ status: "pending", reviewDecision: "" }} />);
    expect(screen.getByText("CI pending")).toBeInTheDocument();
  });

  it("renders 'CI failing' for failing status", () => {
    render(<CIStatusBadge ci={{ status: "failing", reviewDecision: "" }} />);
    expect(screen.getByText("CI failing")).toBeInTheDocument();
  });
});

describe("ReviewBadge", () => {
  it("renders nothing when ci is undefined", () => {
    const { container } = render(<ReviewBadge />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when reviewDecision is empty", () => {
    const { container } = render(
      <ReviewBadge ci={{ status: "passing", reviewDecision: "" }} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders 'Approved' for APPROVED decision", () => {
    render(
      <ReviewBadge ci={{ status: "passing", reviewDecision: "APPROVED" }} />
    );
    expect(screen.getByText("Approved")).toBeInTheDocument();
  });

  it("renders 'Changes requested' for CHANGES_REQUESTED", () => {
    render(
      <ReviewBadge ci={{ status: "passing", reviewDecision: "CHANGES_REQUESTED" }} />
    );
    expect(screen.getByText("Changes requested")).toBeInTheDocument();
  });

  it("renders 'Review required' for REVIEW_REQUIRED", () => {
    render(
      <ReviewBadge ci={{ status: "passing", reviewDecision: "REVIEW_REQUIRED" }} />
    );
    expect(screen.getByText("Review required")).toBeInTheDocument();
  });
});

describe("DiffStats", () => {
  it("renders nothing when both added and deleted are 0", () => {
    const { container } = render(<DiffStats added={0} deleted={0} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders only added when deleted is 0", () => {
    render(<DiffStats added={10} deleted={0} />);
    expect(screen.getByText("+10")).toBeInTheDocument();
    expect(screen.queryByText(/-/)).not.toBeInTheDocument();
  });

  it("renders only deleted when added is 0", () => {
    render(<DiffStats added={0} deleted={5} />);
    expect(screen.getByText("-5")).toBeInTheDocument();
    expect(screen.queryByText(/\+/)).not.toBeInTheDocument();
  });

  it("renders both added and deleted", () => {
    render(<DiffStats added={10} deleted={5} />);
    expect(screen.getByText("+10")).toBeInTheDocument();
    expect(screen.getByText("-5")).toBeInTheDocument();
  });
});

describe("StackStatusBadge", () => {
  it("renders 'Ready to ship' for shippable", () => {
    render(<StackStatusBadge status="shippable" />);
    expect(screen.getByText("Ready to ship")).toBeInTheDocument();
  });

  it("renders 'Needs restack' for pending", () => {
    render(<StackStatusBadge status="pending" />);
    expect(screen.getByText("Needs restack")).toBeInTheDocument();
  });

  it("renders 'Blocked' for blocked", () => {
    render(<StackStatusBadge status="blocked" />);
    expect(screen.getByText("Blocked")).toBeInTheDocument();
  });

  it("renders 'Incomplete' for incomplete", () => {
    render(<StackStatusBadge status="incomplete" />);
    expect(screen.getByText("Incomplete")).toBeInTheDocument();
  });

  it("renders raw status for unknown values", () => {
    render(<StackStatusBadge status="custom-status" />);
    expect(screen.getByText("custom-status")).toBeInTheDocument();
  });
});
