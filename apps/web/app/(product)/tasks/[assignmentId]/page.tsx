import { AssignmentPage } from "../../../../features/html-lessons/HTMLAssignmentPage";

export default async function Page({params}:{params:Promise<{assignmentId:string}>}){
 return <AssignmentPage assignmentID={(await params).assignmentId}/>;
}
