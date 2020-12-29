import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { Comment } from "../models";

@Injectable({ providedIn: 'root' })
export class CommentsService {

    BASE_URL: string = 'http://localhost:8000';

    constructor(private http: HttpClient) {}

    public create(comment: Comment) {
        return this.http.post<any>(this.BASE_URL + `/api/comment`, comment)
            .pipe(
                map(resp => new Comment(resp))
            );
    }

    public list(videoId: number) {
        return this.http.get<any>(this.BASE_URL + '/api/video/' + videoId + '/comments')
            .pipe(
                map(resp => resp.map(comm => new Comment(comm)))
            )
    }

    public get(commId: number) {
        return this.http.get<any>(this.BASE_URL + '/api/comment/' + commId)
            .pipe();
    }

}

