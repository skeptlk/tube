import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";
import { map } from "rxjs/operators";
import { Comment } from "../models";
import { AuthService } from "./auth.service";

@Injectable({ providedIn: 'root' })
export class CommentsService {
    private BASE_URL: string = 'http://localhost:8000';
    public replyTo: Comment;
    public comments: Comment[] = [];

    constructor(
        private http: HttpClient, 
        private auth: AuthService
    ) {}

    public create(text: string, videoId: number) {
        var comm = new Comment();
        comm.text = text;
        comm.videoId = videoId;
        comm.userId = this.auth.currentUserValue.id;
        comm.user = this.auth.currentUserValue;

        if (this.replyTo) {
            comm.replyTo = this.replyTo.id;
            this.replyTo.replies = [comm, ...this.replyTo.replies];
            this.replyTo.replyCount++;
            console.log(this.replyTo.replies, this.replyTo.replyCount);
            
        } else {
            this.comments.push(comm);
        }

        return this.http
            .post<any>(this.BASE_URL + `/api/comment`, comm)
            .pipe(
                map(resp => {
                    var created = new Comment(resp);
                    comm.id = created.id;
                    return created;
                })
            );
    }

    public list(videoId: number) {
        this.comments = [];
        return this.http.get<any>(this.BASE_URL + '/api/video/' + videoId + '/comments')
            .pipe(
                map(resp => {
                    this.comments = resp.map(comm => new Comment(comm));
                    return this.comments;
                })
            )
    }

    public get(commId: number) {
        return this.http.get<any>(this.BASE_URL + '/api/comment/' + commId)
            .pipe(
                map(resp => new Comment(resp))
            );
    }

    public delete(commId: number) {
        return this.http.delete<any>(this.BASE_URL + '/api/comment/' + commId)
            .pipe(
                map(resp => {
                    this.comments = this.comments.filter(comm => comm.id !== commId);
                    return new Comment(resp)
                })
            );
    }

}

