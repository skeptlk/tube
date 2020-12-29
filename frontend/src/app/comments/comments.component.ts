import { Component, Input, OnInit } from '@angular/core';
import { CommentsService } from '../services';
import { Comment, User } from '../models';

@Component({
    selector: 'video-comments',
    templateUrl: './comments.component.html',
    styleUrls: ['./comments.component.scss']
})
export class CommentsComponent implements OnInit {

    @Input() videoId: number;
    @Input() user: User;
    comments: Comment[] = [];
    replyTo: Comment;
    newComment: Comment = new Comment();

    constructor(private commentsService: CommentsService) { }

    ngOnInit() {
        this.commentsService
            .list(this.videoId)
            .subscribe(data => this.comments = data);
    }

    submitComment() {
        const comm = this.newComment;
        comm.videoId = this.videoId;
        comm.userId = this.user.id;
        comm.user = this.user;
        if (this.replyTo) {
            comm.replyTo = this.replyTo.id;
        }
        this.comments.push(comm);

        this.commentsService
            .create(comm)
            .toPromise();
    }

}
