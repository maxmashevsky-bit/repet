import { ReviewSessionPage } from "../../../../../features/learning-reviews/ReviewSessionPage";
export default async function Page({params}:{params:Promise<{sessionId:string}>}){const{sessionId}=await params;return <ReviewSessionPage sessionID={sessionId}/>;}
