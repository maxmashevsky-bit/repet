import {Suspense} from "react";
import {HTMLImporter} from "../../../../../../features/html-lessons/HTMLImporter";
export default function Page(){return <Suspense fallback={<p>Загрузка…</p>}><HTMLImporter/></Suspense>;}
