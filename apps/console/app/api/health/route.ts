export function GET() {
  return Response.json({
    service: "aethergate-console",
    status: "ok",
    timestamp: new Date().toISOString(),
  });
}
