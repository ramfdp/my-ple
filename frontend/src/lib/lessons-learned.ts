export interface LessonLearned {
 id:number; jobId:number; sourceGroup:string; sourceProject:string; sourceRow:number; sourceNumber:string;
 kind:'failure'|'success'; foundOn:string; title:string; location:string;
 internalFactor:string; externalFactor:string; impact:string; action:string;
 status:'open'|'in_progress'|'resolved'; notes:string; resolvedOn:string; version:number;
}

export interface RekapSectionLayout {
 group:string; project:string; headingRow:number; rows:number[];
}
