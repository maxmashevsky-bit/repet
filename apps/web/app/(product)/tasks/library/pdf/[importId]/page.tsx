import { PDFImporterPage } from "../../../../../../features/pdf-import/PDFImporter";
export default async function Page({params}:{params:Promise<{importId:string}>}){const {importId}=await params;return <PDFImporterPage importID={importId}/>;}
