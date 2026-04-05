import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { APP_DISPLAY_NAME, UI_ERROR_HEADING } from "./constants/branding";
import "./styles.css";

type RootErrorBoundaryState = {
  error: string | null;
};

class RootErrorBoundary extends React.Component<React.PropsWithChildren, RootErrorBoundaryState> {
  constructor(props: React.PropsWithChildren) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: unknown): RootErrorBoundaryState {
    const message = error instanceof Error ? error.message : String(error);
    return { error: message };
  }

  componentDidCatch(error: unknown): void {
    console.error("ui runtime error", error);
  }

  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: "24px", color: "#fecaca", fontFamily: "Manrope, Segoe UI, sans-serif" }}>
          <h2 style={{ marginTop: 0 }}>{UI_ERROR_HEADING}</h2>
          <p style={{ marginBottom: "8px" }}>A runtime error occurred while rendering the app.</p>
          <pre style={{ whiteSpace: "pre-wrap", background: "rgba(127, 29, 29, 0.35)", padding: "12px", borderRadius: "8px" }}>{this.state.error}</pre>
        </div>
      );
    }
    return this.props.children;
  }
}

document.title = APP_DISPLAY_NAME;

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <RootErrorBoundary>
      <App />
    </RootErrorBoundary>
  </React.StrictMode>
);
