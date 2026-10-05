import { AcceptInvitationPage } from "../../../../features/auth/AcceptInvitationPage";

export default async function Page({ params }: { params: Promise<{ token: string }> }) {
  const { token } = await params;
  return <AcceptInvitationPage token={token} />;
}
