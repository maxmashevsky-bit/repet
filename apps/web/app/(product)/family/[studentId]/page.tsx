import { FamilyChildPage } from "../../../../features/family/FamilyChildPage";
export default async function Page({params}:{params:Promise<{studentId:string}>}){const{studentId}=await params;return <FamilyChildPage studentID={studentId}/>;}
