import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { Video } from "../models";

@Injectable({ providedIn: 'root' })
export class VideoService {

    constructor(private http: HttpClient) {}

    public BASE_URL = "http://localhost:8000";

    public upload(data: any) {
        return this.http.post<any>(this.BASE_URL + `api/video`, data).pipe();
    }

    public getInfo(id: number) {
        return this.http.get<any>(`http://localhost:8000/v/`+id)
         .pipe(map(resp => new Video(resp)))
    }

}

