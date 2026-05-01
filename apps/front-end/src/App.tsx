import "@mantine/core/styles.css";
import { MantineProvider } from "@mantine/core";
import { useSupabaseAuth } from "./hooks/useSupabaseAuth";
import { ChatPanel } from "./components/ChatPanel";
import { theme } from "./theme";

function App() {
  // Authentication
  const left = useSupabaseAuth("left");
  const right = useSupabaseAuth("right");

  const isLoading = left.isLoading || right.isLoading;
  const error = left.error || right.error;

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  // Ensure we have valid authentication data before rendering
  if (!left.jwt || !left.userId || !right.jwt || !right.userId) {
    return <div>Waiting for authentication...</div>;
  }

  return (
    <MantineProvider theme={theme}>
      <div style={{ 
        display: "flex", 
        height: "100vh", 
        backgroundColor: "#0a0a0a",
        overflow: "hidden"
      }}>
        <div style={{ flex: 1, borderRight: "1px solid #2a2a2a" }}>
          <ChatPanel
            side="left"
            rightUserId={right.userId}
            leftUserId={left.userId}
            jwt={left.jwt}
            userId={left.userId}
          />
        </div>
        <div style={{ flex: 1 }}>
          <ChatPanel
            side="right"
            jwt={right.jwt}
            rightUserId={right.userId}
            leftUserId={left.userId}
            userId={right.userId}
          />
        </div>
      </div>
    </MantineProvider>
  );
}

export default App;
