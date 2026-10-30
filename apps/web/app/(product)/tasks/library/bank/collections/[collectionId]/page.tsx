import { BankCollectionsPage } from "../../../../../../../features/exercise-bank/CollectionsPage";
export default async function Page({params}:{params:Promise<{collectionId:string}>}){const {collectionId}=await params;return <BankCollectionsPage collectionID={collectionId}/>;}
