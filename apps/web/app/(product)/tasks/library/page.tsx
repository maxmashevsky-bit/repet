import { Suspense } from "react";
import { LibraryPage } from "../../../../features/material-library/LibraryPage";
import { LibraryLoading } from "../../../../features/material-library/LibraryUI";

export default function Page() { return <Suspense fallback={<LibraryLoading />}><LibraryPage /></Suspense>; }
